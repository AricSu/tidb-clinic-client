package clinicapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	catalogDataStatusTypeRetained = 1
	catalogDataStatusNeedsRebuild = -1
	catalogDataStatusReadable     = 100
)

type catalogDataStatusItem struct {
	ItemID    string `json:"itemID"`
	Status    int    `json:"status"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	TaskType  int    `json:"taskType"`
}
type catalogDataStatusResponse struct {
	Items []catalogDataStatusItem `json:"items"`
}
type catalogReadiness struct {
	readable      bool
	readableItem  *catalogDataStatusItem
	rebuildItem   *catalogDataStatusItem
	matchedStatus []string
}
type catalogStatusPoller struct {
	client         *catalogClient
	requestContext RequestContext
	item           ClinicDataItem
	trace          requestTrace
	endpoint       string
}

func (c *catalogClient) ListClusterData(ctx context.Context, req ListClusterDataRequest) ([]ClinicDataItem, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "catalog client is nil"}
	}
	if strings.TrimSpace(req.Context.OrgID) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: catalogEndpointPattern, Message: "org id is required"}
	}
	if strings.TrimSpace(req.Context.ClusterID) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: catalogEndpointPattern, Message: "cluster id is required"}
	}
	var resp struct {
		Total     int `json:"total"`
		DataInfos []struct {
			StartTime  int64    `json:"startTime"`
			EndTime    int64    `json:"endTime"`
			ItemID     string   `json:"itemID"`
			Filename   string   `json:"filename"`
			Collectors []string `json:"collectors"`
			HaveLog    bool     `json:"haveLog"`
			HaveMetric bool     `json:"haveMetric"`
			HaveConfig bool     `json:"haveConfig"`
		} `json:"dataInfos"`
	}
	endpoint := catalogEndpoint(req.Context.OrgID, req.Context.ClusterID)
	if err := c.transport.getJSON(ctx, endpoint, nil, nil, requestTraceFromContext(req.Context, ""), &resp); err != nil {
		return nil, err
	}
	items := make([]ClinicDataItem, 0, len(resp.DataInfos))
	for _, item := range resp.DataInfos {
		items = append(items, ClinicDataItem{
			ItemID:     strings.TrimSpace(item.ItemID),
			Filename:   strings.TrimSpace(item.Filename),
			Collectors: append([]string(nil), item.Collectors...),
			HaveLog:    item.HaveLog,
			HaveMetric: item.HaveMetric,
			HaveConfig: item.HaveConfig,
			StartTime:  item.StartTime,
			EndTime:    item.EndTime,
		})
	}
	return items, nil
}
func (c *catalogClient) EnsureCatalogDataReadable(ctx context.Context, requestContext RequestContext, item ClinicDataItem) error {
	if c == nil || c.transport == nil {
		return &Error{Class: ErrBackend, Message: "catalog client is nil"}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(requestContext.OrgID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: catalogDataStatusPattern, Message: "org id is required"}
	}
	if strings.TrimSpace(requestContext.ClusterID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: catalogDataStatusPattern, Message: "cluster id is required"}
	}
	if item.StartTime <= 0 || item.EndTime <= 0 || item.EndTime < item.StartTime {
		return &Error{Class: ErrInvalidRequest, Endpoint: catalogDataStatusPattern, Message: "selected catalog item must include a valid start/end time range"}
	}
	poller := c.newCatalogStatusPoller(requestContext, item)
	triggeredRebuild := false
	probe := 0
	for {
		if err := poller.contextError(ctx); err != nil {
			poller.logLifecycle("rebuild_probe_cancelled", nil, err)
			return err
		}
		resp, err := poller.fetchStatus(ctx)
		if err != nil {
			if ctxErr := poller.contextError(ctx); ctxErr != nil {
				poller.logLifecycle("rebuild_probe_cancelled", nil, ctxErr)
				return ctxErr
			}
			return err
		}
		readiness := assessCatalogReadiness(item, resp.Items)
		if readiness.readable {
			if probe > 0 || triggeredRebuild {
				poller.logLifecycle("rebuild_completed", readiness.readableItem, nil)
			}
			return nil
		}
		if readiness.rebuildItem != nil && !triggeredRebuild {
			poller.logLifecycle("rebuild_required", readiness.rebuildItem, nil)
			if err := poller.triggerRebuild(ctx); err != nil {
				poller.logLifecycle("rebuild_trigger_error", readiness.rebuildItem, err)
				return err
			}
			poller.logLifecycle("rebuild_triggered", readiness.rebuildItem, nil)
			triggeredRebuild = true
		}
		if len(readiness.matchedStatus) == 0 {
			poller.logLifecycle("missing_data_status", nil, nil)
			return &Error{
				Class:    ErrNoData,
				Endpoint: poller.endpoint,
				Message:  fmt.Sprintf("selected catalog item %q has no matching data_status record", strings.TrimSpace(item.ItemID)),
			}
		}
		probe++
		poller.logProbe(readiness.matchedStatus, probe)
		if err := poller.wait(ctx); err != nil {
			poller.logLifecycle("rebuild_probe_cancelled", nil, err)
			return err
		}
	}
}
func matchesCatalogDataStatus(item ClinicDataItem, statusItem catalogDataStatusItem) bool {
	itemID := strings.TrimSpace(item.ItemID)
	statusItemID := strings.TrimSpace(statusItem.ItemID)
	switch {
	case itemID != "" && statusItemID != "":
		return itemID == statusItemID
	default:
		return item.StartTime == statusItem.StartTime && item.EndTime == statusItem.EndTime
	}
}
func assessCatalogReadiness(item ClinicDataItem, statusItems []catalogDataStatusItem) catalogReadiness {
	readiness := catalogReadiness{
		matchedStatus: make([]string, 0, len(statusItems)),
	}
	for i := range statusItems {
		statusItem := statusItems[i]
		if !matchesCatalogDataStatus(item, statusItem) {
			continue
		}
		readiness.matchedStatus = append(readiness.matchedStatus, strconv.Itoa(statusItem.Status))
		if statusItem.Status == catalogDataStatusReadable {
			readiness.readable = true
			readiness.readableItem = &statusItems[i]
			return readiness
		}
		if statusItem.Status == catalogDataStatusNeedsRebuild && readiness.rebuildItem == nil {
			readiness.rebuildItem = &statusItems[i]
		}
	}
	return readiness
}
func (c *catalogClient) newCatalogStatusPoller(requestContext RequestContext, item ClinicDataItem) catalogStatusPoller {
	return catalogStatusPoller{
		client:         c,
		requestContext: requestContext,
		item:           item,
		trace:          requestTraceFromContext(requestContext, item.ItemID),
		endpoint:       catalogDataStatusEndpoint(requestContext.OrgID, requestContext.ClusterID),
	}
}
func (p catalogStatusPoller) triggerRebuild(ctx context.Context) error {
	endpoint := catalogRebuildEndpoint(p.requestContext.OrgID, p.requestContext.ClusterID)
	if err := p.client.transport.putJSON(ctx, endpoint, nil, p.trace, nil, nil); err != nil {
		return err
	}
	return nil
}
func (p catalogStatusPoller) fetchStatus(ctx context.Context) (catalogDataStatusResponse, error) {
	query := url.Values{}
	query.Set("startTime", strconv.FormatInt(p.item.StartTime, 10))
	query.Set("endTime", strconv.FormatInt(p.item.EndTime, 10))
	query.Set("data_type", strconv.Itoa(catalogDataStatusTypeRetained))
	var resp catalogDataStatusResponse
	if err := p.client.transport.getJSON(ctx, p.endpoint, query, nil, p.trace, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}
func (p catalogStatusPoller) probeInterval() time.Duration {
	if p.client == nil || p.client.transport == nil || p.client.transport.rebuildProbeInterval <= 0 {
		return 10 * time.Second
	}
	return p.client.transport.rebuildProbeInterval
}
func (p catalogStatusPoller) wait(ctx context.Context) error {
	timer := time.NewTimer(p.probeInterval())
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return p.contextError(ctx)
	case <-timer.C:
		return nil
	}
}
func (p catalogStatusPoller) contextError(ctx context.Context) error {
	if ctx == nil || ctx.Err() == nil {
		return nil
	}
	switch ctx.Err() {
	case context.DeadlineExceeded:
		return &Error{
			Class:    ErrTimeout,
			Endpoint: p.endpoint,
			Message:  "waiting for tiup-cluster rebuild completion timed out",
			Cause:    ctx.Err(),
		}
	default:
		return &Error{
			Class:    ErrBackend,
			Endpoint: p.endpoint,
			Message:  "waiting for tiup-cluster rebuild completion was cancelled",
			Cause:    ctx.Err(),
		}
	}
}
func (p catalogStatusPoller) logLifecycle(status string, statusItem *catalogDataStatusItem, err error) {
	if p.client == nil || p.client.transport == nil || p.client.transport.logger == nil {
		return
	}
	format := "stage=clinic_catalog endpoint=%s status=%s selected_start_time=%d selected_end_time=%d%s"
	args := []any{p.endpoint, status, p.item.StartTime, p.item.EndTime, p.trace.logSuffix()}
	if statusItem != nil {
		format = "stage=clinic_catalog endpoint=%s status=%s item_status=%d task_type=%d selected_start_time=%d selected_end_time=%d%s"
		args = []any{p.endpoint, status, statusItem.Status, statusItem.TaskType, p.item.StartTime, p.item.EndTime, p.trace.logSuffix()}
	}
	if err != nil {
		format += " err=%q"
		args = append(args, err.Error())
	}
	p.client.transport.logger.Printf(format, args...)
}
func (p catalogStatusPoller) logProbe(matchedStatuses []string, probe int) {
	if p.client == nil || p.client.transport == nil || p.client.transport.logger == nil {
		return
	}
	p.client.transport.logger.Printf(
		"stage=clinic_catalog endpoint=%s status=rebuilding probe=%d probe_interval=%s item_statuses=%s selected_start_time=%d selected_end_time=%d%s",
		p.endpoint,
		probe,
		p.probeInterval(),
		strings.Join(matchedStatuses, ","),
		p.item.StartTime,
		p.item.EndTime,
		p.trace.logSuffix(),
	)
}
func unwrapCollection(raw any) (int, []map[string]any) {
	switch x := raw.(type) {
	case map[string]any:
		total := int(asInt64OrZero(firstPresent(x, "total", "count")))
		switch {
		case x["records"] != nil:
			return total, sliceOfMaps(x["records"])
		case x["items"] != nil:
			return total, sliceOfMaps(x["items"])
		case x["entries"] != nil:
			return total, sliceOfMaps(x["entries"])
		case x["data"] != nil:
			return total, sliceOfMaps(x["data"])
		default:
			return total, []map[string]any{x}
		}
	case []any:
		return 0, sliceOfMaps(x)
	case []map[string]any:
		return 0, x
	default:
		return 0, nil
	}
}
