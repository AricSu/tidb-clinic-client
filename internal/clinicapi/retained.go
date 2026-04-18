package clinicapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (c *catalogClient) QuerySlowQueries(ctx context.Context, req SlowQueryRequest) (SlowQueryResult, error) {
	if c == nil || c.transport == nil {
		return SlowQueryResult{}, &Error{Class: ErrBackend, Message: "catalog client is nil"}
	}
	endpoint := slowQueriesClusterEndpoint(req.Context.OrgID, req.Context.ClusterID)
	route, err := routeFromSlowQueryContext(endpoint, req.Context, req.ItemID)
	if err != nil {
		return SlowQueryResult{}, err
	}
	if req.StartTime <= 0 || req.EndTime <= 0 || req.EndTime < req.StartTime {
		return SlowQueryResult{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "valid start/end time range is required"}
	}
	query := url.Values{}
	if itemID := strings.TrimSpace(req.ItemID); itemID != "" {
		query.Set("itemID", itemID)
	}
	query.Set("startTime", strconv.FormatInt(req.StartTime, 10))
	query.Set("endTime", strconv.FormatInt(req.EndTime, 10))
	if orderBy := strings.TrimSpace(req.OrderBy); orderBy != "" {
		query.Set("orderBy", orderBy)
	}
	if req.Desc {
		query.Set("desc", "true")
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	var raw any
	if err := c.transport.getJSON(ctx, endpoint, query, route.headers, route.trace, &raw); err != nil {
		return SlowQueryResult{}, err
	}
	return decodeSlowQueryResult(raw), nil
}

func (c *catalogClient) QuerySlowQuerySamples(ctx context.Context, req SlowQuerySamplesRequest) (SlowQuerySamplesResult, error) {
	if c == nil || c.transport == nil {
		return SlowQuerySamplesResult{}, &Error{Class: ErrBackend, Message: "catalog client is nil"}
	}
	endpoint := slowQueriesClusterEndpoint(req.Context.OrgID, req.Context.ClusterID)
	route, err := routeFromSlowQueryContext(endpoint, req.Context, req.ItemID)
	if err != nil {
		return SlowQuerySamplesResult{}, err
	}
	if req.StartTime <= 0 || req.EndTime <= 0 || req.EndTime < req.StartTime {
		return SlowQuerySamplesResult{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "valid start/end time range is required"}
	}
	query := url.Values{}
	if itemID := strings.TrimSpace(req.ItemID); itemID != "" {
		query.Set("itemID", itemID)
	}
	query.Set("startTime", strconv.FormatInt(req.StartTime, 10))
	query.Set("endTime", strconv.FormatInt(req.EndTime, 10))
	if digest := strings.TrimSpace(req.Digest); digest != "" {
		query.Set("digest", digest)
	}
	if orderBy := strings.TrimSpace(req.OrderBy); orderBy != "" {
		query.Set("orderBy", orderBy)
	}
	if req.Desc {
		query.Set("desc", "true")
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	if len(req.Fields) > 0 {
		fields := make([]string, 0, len(req.Fields))
		for _, item := range req.Fields {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				fields = append(fields, trimmed)
			}
		}
		if len(fields) > 0 {
			query.Set("fields", strings.Join(fields, ","))
		}
	}
	var raw any
	if err := c.transport.getJSON(ctx, endpoint, query, route.headers, route.trace, &raw); err != nil {
		return SlowQuerySamplesResult{}, err
	}
	return decodeSlowQuerySamplesResult(raw, req.ItemID), nil
}

func decodeSlowQueryResult(raw any) SlowQueryResult {
	switch value := raw.(type) {
	case []any:
		return slowQueryResultFromMaps(sliceOfMaps(value), len(value))
	case []map[string]any:
		return slowQueryResultFromMaps(value, len(value))
	case map[string]any:
		total, items := unwrapCollection(value)
		if value["slowQueries"] != nil {
			items = sliceOfMaps(value["slowQueries"])
		}
		return slowQueryResultFromMaps(items, total)
	default:
		return SlowQueryResult{}
	}
}

func decodeSlowQuerySamplesResult(raw any, itemID string) SlowQuerySamplesResult {
	total, items := unwrapCollection(raw)
	switch value := raw.(type) {
	case []any:
		items = sliceOfMaps(value)
		if total == 0 {
			total = len(value)
		}
	case []map[string]any:
		items = value
		if total == 0 {
			total = len(value)
		}
	case map[string]any:
		switch {
		case value["slowQueries"] != nil:
			items = sliceOfMaps(value["slowQueries"])
		case value["items"] != nil:
			items = sliceOfMaps(value["items"])
		}
	default:
		return SlowQuerySamplesResult{}
	}
	if total == 0 {
		total = len(items)
	}
	out := SlowQuerySamplesResult{
		Total: total,
		Items: make([]map[string]any, 0, len(items)),
	}
	for _, item := range items {
		next := cloneAnyMap(item)
		if next == nil {
			next = map[string]any{}
		}
		if asTrimmedString(firstPresent(next, "item_id", "itemID")) == "" && strings.TrimSpace(itemID) != "" {
			next["item_id"] = strings.TrimSpace(itemID)
		}
		if asTrimmedString(firstPresent(next, "source_ref", "sourceRef")) == "" && strings.TrimSpace(itemID) != "" {
			next["source_ref"] = strings.TrimSpace(itemID)
		}
		out.Items = append(out.Items, next)
	}
	return out
}

func slowQueryResultFromMaps(items []map[string]any, total int) SlowQueryResult {
	if total == 0 {
		total = len(items)
	}
	out := SlowQueryResult{
		Total:   total,
		Records: make([]SlowQueryRecord, 0, len(items)),
	}
	for _, item := range items {
		out.Records = append(out.Records, SlowQueryRecord{
			Digest:     asTrimmedString(firstPresent(item, "digest")),
			SQLText:    asTrimmedString(firstPresent(item, "query", "sql")),
			QueryTime:  asFloat64OrZero(firstPresent(item, "queryTime", "query_time")),
			ExecCount:  asInt64OrZero(firstPresent(item, "execCount", "exec_count", "request_count")),
			User:       asTrimmedString(firstPresent(item, "user")),
			DB:         asTrimmedString(firstPresent(item, "db")),
			TableNames: sliceOfStrings(firstPresent(item, "tableNames", "table_names")),
			IndexNames: sliceOfStrings(firstPresent(item, "indexNames", "index_names")),
			SourceRef:  asTrimmedString(firstPresent(item, "sourceRef", "source_ref", "packageid", "packageID")),
		})
	}
	return out
}

func routeFromSlowQueryContext(endpoint string, ctx RequestContext, itemID string) (requestRoute, error) {
	if strings.EqualFold(strings.TrimSpace(ctx.RoutingOrgType), "cloud") || strings.EqualFold(strings.TrimSpace(ctx.OrgType), "cloud") {
		return routeFromContext(endpoint, ctx, itemID)
	}
	return routeFromItemContext(endpoint, ctx, itemID)
}

const (
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
	dataType       CatalogDataType
	rebuildStatus  map[int]bool
	trace          requestTrace
	endpoint       string
}
type catalogRebuildRequest struct {
	ItemID    string `json:"itemID,omitempty"`
	StartTime int64  `json:"startTime,omitempty"`
	EndTime   int64  `json:"endTime,omitempty"`
	DataType  int    `json:"dataType,omitempty"`
	TaskType  int    `json:"taskType,omitempty"`
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
func (c *catalogClient) DownloadCollectedData(ctx context.Context, req CollectedDataDownloadRequest) ([]byte, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "catalog client is nil"}
	}
	route, err := routeFromItemContext(catalogDownloadPattern, req.Context, req.ItemID)
	if err != nil {
		return nil, err
	}
	endpoint := catalogDownloadEndpoint(req.Context.OrgID, req.Context.ClusterID, req.ItemID)
	return c.transport.getBytes(ctx, endpoint, nil, route.headers, route.trace)
}
func (c *catalogClient) EnsureCatalogDataReadable(ctx context.Context, req EnsureCatalogDataReadableRequest) error {
	if c == nil || c.transport == nil {
		return &Error{Class: ErrBackend, Message: "catalog client is nil"}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	requestContext := req.Context
	item := req.Item
	if strings.TrimSpace(requestContext.OrgID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: catalogDataStatusPattern, Message: "org id is required"}
	}
	if strings.TrimSpace(requestContext.ClusterID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: catalogDataStatusPattern, Message: "cluster id is required"}
	}
	if item.StartTime <= 0 || item.EndTime <= 0 || item.EndTime < item.StartTime {
		return &Error{Class: ErrInvalidRequest, Endpoint: catalogDataStatusPattern, Message: "selected catalog item must include a valid start/end time range"}
	}
	dataType, rebuildStatuses, err := catalogStatusSpec(req.DataType)
	if err != nil {
		return err
	}
	poller := c.newCatalogStatusPoller(requestContext, item, dataType, rebuildStatuses)
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
		readiness := assessCatalogReadiness(item, resp.Items, poller.rebuildStatus)
		if readiness.readable {
			if probe > 0 || triggeredRebuild {
				poller.logLifecycle("rebuild_completed", readiness.readableItem, nil)
			}
			return nil
		}
		if readiness.rebuildItem != nil && !triggeredRebuild {
			poller.logLifecycle("rebuild_required", readiness.rebuildItem, nil)
			if err := poller.triggerRebuild(ctx, readiness.rebuildItem); err != nil {
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
func catalogStatusSpec(dataType CatalogDataType) (CatalogDataType, map[int]bool, error) {
	switch dataType {
	case 0, CatalogDataTypeRetained:
		return CatalogDataTypeRetained, map[int]bool{
			0:                             true,
			catalogDataStatusNeedsRebuild: true,
		}, nil
	case CatalogDataTypeLogs:
		return CatalogDataTypeLogs, map[int]bool{
			0:                             true,
			catalogDataStatusNeedsRebuild: true,
		}, nil
	case CatalogDataTypeCollectedDownload:
		return CatalogDataTypeCollectedDownload, map[int]bool{
			0:                             true,
			catalogDataStatusNeedsRebuild: true,
		}, nil
	default:
		return 0, nil, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: catalogDataStatusPattern,
			Message:  fmt.Sprintf("unsupported catalog data_type=%d", dataType),
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
func assessCatalogReadiness(item ClinicDataItem, statusItems []catalogDataStatusItem, rebuildStatuses map[int]bool) catalogReadiness {
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
		if rebuildStatuses[statusItem.Status] && readiness.rebuildItem == nil {
			readiness.rebuildItem = &statusItems[i]
		}
	}
	return readiness
}
func (c *catalogClient) newCatalogStatusPoller(requestContext RequestContext, item ClinicDataItem, dataType CatalogDataType, rebuildStatuses map[int]bool) catalogStatusPoller {
	return catalogStatusPoller{
		client:         c,
		requestContext: requestContext,
		item:           item,
		dataType:       dataType,
		rebuildStatus:  rebuildStatuses,
		trace:          requestTraceFromContext(requestContext, item.ItemID),
		endpoint:       catalogDataStatusEndpoint(requestContext.OrgID, requestContext.ClusterID),
	}
}
func (p catalogStatusPoller) triggerRebuild(ctx context.Context, statusItem *catalogDataStatusItem) error {
	endpoint := catalogRebuildEndpoint(p.requestContext.OrgID, p.requestContext.ClusterID)
	payload, err := p.rebuildRequest(statusItem)
	if err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "failed to encode rebuild request body", Cause: err}
	}
	return p.client.transport.doJSON(ctx, requestOptions{
		method:      http.MethodPut,
		path:        endpoint,
		body:        body,
		contentType: "application/json",
		trace:       p.trace,
	}, nil)
}
func (p catalogStatusPoller) rebuildRequest(statusItem *catalogDataStatusItem) (catalogRebuildRequest, error) {
	itemID := strings.TrimSpace(p.item.ItemID)
	startTime := p.item.StartTime
	endTime := p.item.EndTime
	taskType := 0
	if statusItem != nil {
		if value := strings.TrimSpace(statusItem.ItemID); value != "" {
			itemID = value
		}
		if statusItem.StartTime > 0 {
			startTime = statusItem.StartTime
		}
		if statusItem.EndTime > 0 {
			endTime = statusItem.EndTime
		}
		taskType = statusItem.TaskType
	}
	if itemID != "" {
		return catalogRebuildRequest{
			ItemID:   itemID,
			DataType: int(p.dataType),
		}, nil
	}
	if startTime <= 0 || endTime <= 0 {
		return catalogRebuildRequest{}, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: catalogRebuildPattern,
			Message:  "rebuild request requires item id or valid start/end time range",
		}
	}
	return catalogRebuildRequest{
		ItemID:    itemID,
		StartTime: startTime,
		EndTime:   endTime,
		DataType:  int(p.dataType),
		TaskType:  taskType,
	}, nil
}
func (p catalogStatusPoller) fetchStatus(ctx context.Context) (catalogDataStatusResponse, error) {
	query := url.Values{}
	query.Set("startTime", strconv.FormatInt(p.item.StartTime, 10))
	query.Set("endTime", strconv.FormatInt(p.item.EndTime, 10))
	query.Set("data_type", strconv.Itoa(int(p.dataType)))
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
	format := "stage=clinic_catalog endpoint=%s status=%s data_type=%d selected_start_time=%d selected_end_time=%d%s"
	args := []any{p.endpoint, status, p.dataType, p.item.StartTime, p.item.EndTime, p.trace.logSuffix()}
	if statusItem != nil {
		format = "stage=clinic_catalog endpoint=%s status=%s data_type=%d item_status=%d task_type=%d selected_start_time=%d selected_end_time=%d%s"
		args = []any{p.endpoint, status, p.dataType, statusItem.Status, statusItem.TaskType, p.item.StartTime, p.item.EndTime, p.trace.logSuffix()}
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
		"stage=clinic_catalog endpoint=%s status=rebuilding data_type=%d probe=%d probe_interval=%s item_statuses=%s selected_start_time=%d selected_end_time=%d%s",
		p.endpoint,
		p.dataType,
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
