package clinicapi

import (
	"context"
	"strings"
)

// ListClusterData returns the uploaded diagnostic bundles for a cluster.
func (c *CatalogClient) ListClusterData(ctx context.Context, req ListClusterDataRequest) ([]ClinicDataItem, error) {
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
