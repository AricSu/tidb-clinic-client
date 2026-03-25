package clinicapi

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// GetCluster resolves cloud routing metadata for a known cluster identifier.
func (c *CloudClient) GetCluster(ctx context.Context, req CloudClusterLookupRequest) (CloudCluster, error) {
	if c == nil || c.transport == nil {
		return CloudCluster{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudClusterLookup(cloudClusterLookupPath, req)
	if err != nil {
		return CloudCluster{}, err
	}
	query := url.Values{}
	query.Set("cluster_id", strings.TrimSpace(req.ClusterID))
	query.Set("show_deleted", strconv.FormatBool(req.ShowDeleted))
	query.Set("limit", "1")
	query.Set("page", "1")

	var resp struct {
		Items []struct {
			ClusterID  string `json:"clusterID"`
			Name       string `json:"clusterName"`
			Provider   string `json:"clusterProviderName"`
			Region     string `json:"clusterRegionName"`
			DeployType string `json:"clusterDeployType"`
			OrgID      string `json:"orgID"`
			TenantID   string `json:"tenantID"`
			ProjectID  string `json:"projectID"`
			CreatedAt  int64  `json:"clusterCreatedAt"`
			DeletedAt  *int64 `json:"clusterDeletedAt"`
			Status     string `json:"clusterStatus"`
		} `json:"items"`
	}
	if err := c.transport.getJSON(ctx, cloudClusterLookupEndpoint(), query, route.headers, route.trace, &resp); err != nil {
		return CloudCluster{}, err
	}
	if len(resp.Items) == 0 {
		return CloudCluster{}, &Error{Class: ErrNotFound, Endpoint: cloudClusterLookupPath, Message: "cloud cluster not found"}
	}
	item := resp.Items[0]
	return CloudCluster{
		ClusterID:  strings.TrimSpace(item.ClusterID),
		Name:       strings.TrimSpace(item.Name),
		Provider:   strings.TrimSpace(item.Provider),
		Region:     strings.TrimSpace(item.Region),
		DeployType: strings.TrimSpace(item.DeployType),
		OrgID:      strings.TrimSpace(item.OrgID),
		TenantID:   strings.TrimSpace(item.TenantID),
		ProjectID:  strings.TrimSpace(item.ProjectID),
		CreatedAt:  item.CreatedAt,
		DeletedAt:  item.DeletedAt,
		Status:     strings.TrimSpace(item.Status),
	}, nil
}

// GetClusterDetail returns the detail object for a known cloud target.
func (c *CloudClient) GetClusterDetail(ctx context.Context, req CloudClusterDetailRequest) (CloudClusterDetail, error) {
	if c == nil || c.transport == nil {
		return CloudClusterDetail{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudTarget(clusterDetailPattern, req.Target)
	if err != nil {
		return CloudClusterDetail{}, err
	}
	var resp struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Components map[CloudClusterComponentType]struct {
			Replicas            int    `json:"replicas"`
			TierName            string `json:"tierName"`
			StorageInstanceType string `json:"storageInstanceType"`
			Storages            *struct {
				Data *struct {
					IOPS int64 `json:"iops"`
				} `json:"data"`
			} `json:"storages"`
		} `json:"components"`
	}
	endpoint := clusterDetailEndpoint(req.Target.OrgID, req.Target.ClusterID)
	if err := c.transport.getJSON(ctx, endpoint, nil, route.headers, route.trace, &resp); err != nil {
		return CloudClusterDetail{}, err
	}
	out := CloudClusterDetail{
		ID:         strings.TrimSpace(resp.ID),
		Name:       strings.TrimSpace(resp.Name),
		Components: make(map[CloudClusterComponentType]CloudClusterComponent, len(resp.Components)),
	}
	for kind, component := range resp.Components {
		next := CloudClusterComponent{
			Replicas:            component.Replicas,
			TierName:            strings.TrimSpace(component.TierName),
			StorageInstanceType: strings.TrimSpace(component.StorageInstanceType),
		}
		if component.Storages != nil && component.Storages.Data != nil {
			next.StorageIOPS = component.Storages.Data.IOPS
		}
		out.Components[kind] = next
	}
	return out, nil
}

// QueryEvents returns cloud activity events for a known target and time range.
func (c *CloudClient) QueryEvents(ctx context.Context, req CloudEventsRequest) (CloudEventsResult, error) {
	if c == nil || c.transport == nil {
		return CloudEventsResult{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudTarget(cloudEventsPattern, req.Target)
	if err != nil {
		return CloudEventsResult{}, err
	}
	if req.StartTime <= 0 || req.EndTime <= 0 || req.EndTime < req.StartTime {
		return CloudEventsResult{}, &Error{Class: ErrInvalidRequest, Endpoint: cloudEventsPattern, Message: "valid start/end time range is required"}
	}
	query := url.Values{}
	query.Set("begin_ts", strconv.FormatInt(req.StartTime, 10))
	query.Set("end_ts", strconv.FormatInt(req.EndTime, 10))
	var resp struct {
		Total      int `json:"total"`
		Activities []struct {
			EventID         string         `json:"event_id"`
			Name            string         `json:"name"`
			DisplayName     string         `json:"display_name"`
			CalibrationTime int64          `json:"calibration_time"`
			Payload         map[string]any `json:"payload"`
		} `json:"activities"`
	}
	endpoint := cloudEventsEndpoint(req.Target.OrgID, req.Target.ClusterID)
	if err := c.transport.getJSON(ctx, endpoint, query, route.headers, route.trace, &resp); err != nil {
		return CloudEventsResult{}, err
	}
	out := CloudEventsResult{Total: resp.Total, Events: make([]CloudClusterEvent, 0, len(resp.Activities))}
	for _, event := range resp.Activities {
		out.Events = append(out.Events, CloudClusterEvent{
			EventID:     strings.TrimSpace(event.EventID),
			Name:        strings.TrimSpace(event.Name),
			DisplayName: strings.TrimSpace(event.DisplayName),
			CreateTime:  event.CalibrationTime,
			Payload:     event.Payload,
		})
	}
	return out, nil
}

// GetEventDetail returns payload.detail for a known cloud event.
func (c *CloudClient) GetEventDetail(ctx context.Context, req CloudEventDetailRequest) (map[string]any, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudTarget(cloudEventsPattern, req.Target)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.EventID) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: cloudEventsPattern, Message: "event id is required"}
	}
	var resp struct {
		Payload struct {
			Detail map[string]any `json:"detail"`
		} `json:"payload"`
	}
	endpoint := cloudEventDetailEndpoint(req.Target.OrgID, req.Target.ClusterID, req.EventID)
	if err := c.transport.getJSON(ctx, endpoint, nil, route.headers, route.trace, &resp); err != nil {
		return nil, err
	}
	if resp.Payload.Detail == nil {
		return map[string]any{}, nil
	}
	return resp.Payload.Detail, nil
}

// GetTopSQLSummary returns the cloud NGM TopSQL summary for a known target and
// component instance.
func (c *CloudClient) GetTopSQLSummary(ctx context.Context, req CloudTopSQLSummaryRequest) ([]CloudTopSQL, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudNGMTarget(ngmTopSQLEndpoint, req.Target)
	if err != nil {
		return nil, err
	}
	component := strings.TrimSpace(req.Component)
	if component == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmTopSQLEndpoint, Message: "component is required"}
	}
	instance := strings.TrimSpace(req.Instance)
	if instance == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmTopSQLEndpoint, Message: "instance is required"}
	}
	start := strings.TrimSpace(req.Start)
	end := strings.TrimSpace(req.End)
	if start == "" || end == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmTopSQLEndpoint, Message: "start and end are required"}
	}

	query := url.Values{}
	query.Set("start", start)
	query.Set("end", end)
	query.Set("instance", ngmInstance(component, instance, req.Target.ClusterID))
	query.Set("instance_type", component)
	query.Set("top", strconv.Itoa(defaultInt(req.Top, 5)))
	query.Set("window", defaultString(req.Window, "60s"))
	query.Set("group_by", defaultString(req.GroupBy, "query"))

	var resp struct {
		Data []struct {
			SQLDigest         string  `json:"sql_digest"`
			SQLText           string  `json:"sql_text"`
			CPUTimeMS         float64 `json:"cpu_time_ms"`
			ExecCountPerSec   float64 `json:"exec_count_per_sec"`
			DurationPerExecMS float64 `json:"duration_per_exec_ms"`
			ScanRecordsPerSec float64 `json:"scan_records_per_sec"`
			ScanIndexesPerSec float64 `json:"scan_indexes_per_sec"`
			Plans             []struct {
				PlanDigest        string  `json:"plan_digest"`
				PlanText          string  `json:"plan_text"`
				TimestampSec      []int64 `json:"timestamp_sec"`
				CPUTimeMS         []int64 `json:"cpu_time_ms"`
				ExecCountPerSec   float64 `json:"exec_count_per_sec"`
				DurationPerExecMS float64 `json:"duration_per_exec_ms"`
				ScanRecordsPerSec float64 `json:"scan_records_per_sec"`
				ScanIndexesPerSec float64 `json:"scan_indexes_per_sec"`
			} `json:"plans"`
		} `json:"data"`
	}
	if err := c.transport.getJSON(ctx, ngmTopSQLEndpoint, query, route.headers, route.trace, &resp); err != nil {
		return nil, err
	}
	out := make([]CloudTopSQL, 0, len(resp.Data))
	for _, sql := range resp.Data {
		next := CloudTopSQL{
			SQLDigest:         strings.TrimSpace(sql.SQLDigest),
			SQLText:           strings.TrimSpace(sql.SQLText),
			CPUTimeMS:         sql.CPUTimeMS,
			ExecCountPerSec:   sql.ExecCountPerSec,
			DurationPerExecMS: sql.DurationPerExecMS,
			ScanRecordsPerSec: sql.ScanRecordsPerSec,
			ScanIndexesPerSec: sql.ScanIndexesPerSec,
			Plans:             make([]CloudTopSQLPlan, 0, len(sql.Plans)),
		}
		for _, plan := range sql.Plans {
			next.Plans = append(next.Plans, CloudTopSQLPlan{
				PlanDigest:        strings.TrimSpace(plan.PlanDigest),
				PlanText:          strings.TrimSpace(plan.PlanText),
				TimestampSec:      append([]int64(nil), plan.TimestampSec...),
				CPUTimeMS:         append([]int64(nil), plan.CPUTimeMS...),
				ExecCountPerSec:   plan.ExecCountPerSec,
				DurationPerExecMS: plan.DurationPerExecMS,
				ScanRecordsPerSec: plan.ScanRecordsPerSec,
				ScanIndexesPerSec: plan.ScanIndexesPerSec,
			})
		}
		out = append(out, next)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CPUTimeMS > out[j].CPUTimeMS })
	return out, nil
}

// GetTopSlowQueries returns the cloud NGM slow-query aggregate summary for a
// known target.
func (c *CloudClient) GetTopSlowQueries(ctx context.Context, req CloudTopSlowQueriesRequest) ([]CloudTopSlowQuery, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudNGMTarget(ngmTopSlowQueriesPath, req.Target)
	if err != nil {
		return nil, err
	}
	start := strings.TrimSpace(req.Start)
	if start == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmTopSlowQueriesPath, Message: "start is required"}
	}
	query := url.Values{}
	query.Set("begin_time", start)
	query.Set("hours", strconv.Itoa(defaultInt(req.Hours, 1)))
	query.Set("order_by", defaultString(req.OrderBy, "sum_latency"))
	query.Set("limit", strconv.Itoa(defaultInt(req.Limit, 10)))

	var resp []map[string]any
	if err := c.transport.getJSON(ctx, ngmTopSlowQueriesPath, query, route.headers, route.trace, &resp); err != nil {
		return nil, err
	}
	out := make([]CloudTopSlowQuery, 0, len(resp))
	for _, row := range resp {
		out = append(out, CloudTopSlowQuery{
			DB:            asTrimmedString(row["db"]),
			SQLDigest:     asTrimmedString(row["sql_digest"]),
			SQLText:       asTrimmedString(row["sql_text"]),
			StatementType: asTrimmedString(row["statement_type"]),
			Count:         asInt64OrZero(row["count"]),
			SumLatency:    asFloat64OrZero(row["sum_latency"]),
			MaxLatency:    asFloat64OrZero(row["max_latency"]),
			AvgLatency:    asFloat64OrZero(row["avg_latency"]),
			SumMemory:     asFloat64OrZero(row["sum_memory"]),
			MaxMemory:     asFloat64OrZero(row["max_memory"]),
			AvgMemory:     asFloat64OrZero(row["avg_memory"]),
			SumDisk:       asFloat64OrZero(row["sum_disk"]),
			MaxDisk:       asFloat64OrZero(row["max_disk"]),
			AvgDisk:       asFloat64OrZero(row["avg_disk"]),
			Detail:        cloneAnyMap(asAnyMap(row["detail"])),
		})
	}
	return out, nil
}

// ListSlowQueries returns slow-query samples for one digest from cloud NGM.
func (c *CloudClient) ListSlowQueries(ctx context.Context, req CloudSlowQueryListRequest) ([]CloudSlowQueryListEntry, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudNGMTarget(ngmSlowQueryListPath, req.Target)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Digest) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryListPath, Message: "digest is required"}
	}
	if strings.TrimSpace(req.Start) == "" || strings.TrimSpace(req.End) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryListPath, Message: "start and end are required"}
	}
	query := url.Values{}
	query.Set("begin_time", strings.TrimSpace(req.Start))
	query.Set("end_time", strings.TrimSpace(req.End))
	query.Set("digest", strings.TrimSpace(req.Digest))
	query.Set("order_by", defaultString(req.OrderBy, "query_time"))
	query.Set("limit", strconv.Itoa(defaultInt(req.Limit, 10)))
	query.Set("desc", strconv.FormatBool(true))
	query.Set("fields", strings.Join(defaultFields(req.Fields), ","))

	var resp []map[string]any
	if err := c.transport.getJSON(ctx, ngmSlowQueryListPath, query, route.headers, route.trace, &resp); err != nil {
		return nil, err
	}
	out := make([]CloudSlowQueryListEntry, 0, len(resp))
	for _, row := range resp {
		out = append(out, CloudSlowQueryListEntry{
			Digest:       asTrimmedString(row["digest"]),
			Query:        asTrimmedString(row["query"]),
			Timestamp:    asTrimmedString(row["timestamp"]),
			QueryTime:    asFloat64OrZero(row["query_time"]),
			MemoryMax:    asFloat64OrZero(row["memory_max"]),
			RequestCount: asInt64OrZero(row["request_count"]),
			ConnectionID: asTrimmedString(firstPresent(row, "connection_id", "connect_id")),
			Raw:          cloneAnyMap(row),
		})
	}
	return out, nil
}

// GetSlowQueryDetail returns one detailed slow-query record from cloud NGM.
func (c *CloudClient) GetSlowQueryDetail(ctx context.Context, req CloudSlowQueryDetailRequest) (map[string]any, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudNGMTarget(ngmSlowQueryDetailPath, req.Target)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Digest) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryDetailPath, Message: "digest is required"}
	}
	if strings.TrimSpace(req.ConnectionID) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryDetailPath, Message: "connection id is required"}
	}
	if strings.TrimSpace(req.Timestamp) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryDetailPath, Message: "timestamp is required"}
	}
	query := url.Values{}
	query.Set("digest", strings.TrimSpace(req.Digest))
	query.Set("connect_id", strings.TrimSpace(req.ConnectionID))
	query.Set("timestamp", strings.TrimSpace(req.Timestamp))

	var resp map[string]any
	if err := c.transport.getJSON(ctx, ngmSlowQueryDetailPath, query, route.headers, route.trace, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return map[string]any{}, nil
	}
	return resp, nil
}

func defaultInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func defaultString(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func defaultFields(fields []string) []string {
	if len(fields) == 0 {
		return []string{"query", "timestamp", "query_time", "memory_max", "request_count", "digest", "connection_id"}
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{"query", "timestamp", "query_time", "memory_max", "request_count", "digest", "connection_id"}
	}
	return out
}

func ngmInstance(component, instance, clusterID string) string {
	port := "10080"
	if strings.EqualFold(strings.TrimSpace(component), "tikv") {
		port = "20160"
	}
	return strings.TrimSpace(instance) + ".db-" + strings.TrimSpace(component) + "-peer.tidb" + strings.TrimSpace(clusterID) + ".svc:" + port
}

func asTrimmedString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func asFloat64OrZero(v any) float64 {
	if f, ok := asFloat64(v); ok {
		return f
	}
	return 0
}

func asInt64OrZero(v any) int64 {
	if n, ok := asInt64(v); ok {
		return n
	}
	return 0
}

func asAnyMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func firstPresent(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return nil
}
