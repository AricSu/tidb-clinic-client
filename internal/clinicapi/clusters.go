package clinicapi

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

func (c *cloudClient) SearchClusters(ctx context.Context, req CloudClusterSearchRequest) ([]CloudCluster, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	if strings.TrimSpace(req.Query) == "" && strings.TrimSpace(req.ClusterID) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: clusterLookupPath, Message: "query or cluster id is required"}
	}
	route := requestRoute{
		trace: requestTrace{
			clusterID: strings.TrimSpace(req.ClusterID),
		},
	}
	query := clusterLookupQueryValues(clusterLookupQueryText(req.Query, req.ClusterID), req.ShowDeleted, defaultInt(req.Limit, 10), defaultInt(req.Page, 1))
	var resp struct {
		Items []clusterLookupItem `json:"items"`
	}
	if err := c.transport.getJSON(ctx, clusterLookupEndpoint(), query, nil, route.trace, &resp); err != nil {
		return nil, err
	}
	out := make([]CloudCluster, 0, len(resp.Items))
	for _, item := range resp.Items {
		out = append(out, decodeClusterLookupItem(item))
	}
	return out, nil
}
func (c *cloudClient) GetCluster(ctx context.Context, req CloudClusterLookupRequest) (CloudCluster, error) {
	if c == nil || c.transport == nil {
		return CloudCluster{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromClusterLookup(clusterLookupPath, req)
	if err != nil {
		return CloudCluster{}, err
	}
	query := clusterLookupQueryValues(strings.TrimSpace(req.ClusterID), req.ShowDeleted, 10, 1)
	var resp struct {
		Items []clusterLookupItem `json:"items"`
	}
	if err := c.transport.getJSON(ctx, clusterLookupEndpoint(), query, route.headers, route.trace, &resp); err != nil {
		return CloudCluster{}, err
	}
	exactMatches := make([]CloudCluster, 0, len(resp.Items))
	for _, item := range resp.Items {
		cluster := decodeClusterLookupItem(item)
		if cluster.ClusterID == strings.TrimSpace(req.ClusterID) {
			exactMatches = append(exactMatches, cluster)
		}
	}
	switch len(exactMatches) {
	case 0:
		return CloudCluster{}, &Error{Class: ErrNotFound, Endpoint: clusterLookupPath, Message: "cluster not found"}
	case 1:
		return exactMatches[0], nil
	default:
		return CloudCluster{}, &Error{Class: ErrInvalidRequest, Endpoint: clusterLookupPath, Message: "cluster lookup returned multiple exact matches"}
	}
}
func clusterLookupQueryText(queryText, clusterID string) string {
	if trimmed := strings.TrimSpace(clusterID); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(queryText)
}
func clusterLookupQueryValues(queryText string, showDeleted bool, limit, page int) url.Values {
	query := url.Values{}
	query.Set("query", strings.TrimSpace(queryText))
	query.Set("show_deleted", strconv.FormatBool(showDeleted))
	query.Set("sort", "")
	query.Set("order", "")
	query.Set("limit", strconv.Itoa(limit))
	query.Set("page", strconv.Itoa(page))
	return query
}
func (c *cloudClient) GetOrg(ctx context.Context, req OrgRequest) (Org, error) {
	if c == nil || c.transport == nil {
		return Org{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	if strings.TrimSpace(req.OrgID) == "" {
		return Org{}, &Error{Class: ErrInvalidRequest, Endpoint: orgDetailPattern, Message: "org id is required"}
	}
	endpoint := orgDetailEndpoint(req.OrgID)
	var resp map[string]any
	if err := c.transport.getJSON(ctx, endpoint, nil, nil, requestTrace{
		orgID: strings.TrimSpace(req.OrgID),
	}, &resp); err != nil {
		return Org{}, err
	}
	return Org{
		ID:   asTrimmedString(firstPresent(resp, "id", "orgID", "orgId")),
		Name: asTrimmedString(firstPresent(resp, "name", "orgName")),
		Type: asTrimmedString(firstPresent(resp, "type", "orgType")),
	}, nil
}
func (c *cloudClient) GetClusterDetail(ctx context.Context, req CloudClusterDetailRequest) (CloudClusterDetail, error) {
	if c == nil || c.transport == nil {
		return CloudClusterDetail{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromSharedControlPlaneTarget(clusterDetailPattern, req.Target, req.TraceOrgType)
	if err != nil {
		return CloudClusterDetail{}, err
	}
	var resp map[string]any
	endpoint := clusterDetailEndpoint(req.Target.OrgID, req.Target.ClusterID)
	if err := c.transport.getJSON(ctx, endpoint, nil, route.headers, route.trace, &resp); err != nil {
		return CloudClusterDetail{}, err
	}
	return decodeCloudClusterDetail(resp), nil
}
func (c *cloudClient) GetResourcePoolComponents(ctx context.Context, req CloudResourcePoolComponentsRequest) (CloudClusterDetail, error) {
	if c == nil || c.transport == nil {
		return CloudClusterDetail{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudTarget(resourcePoolPattern, req.Target)
	if err != nil {
		return CloudClusterDetail{}, err
	}
	var resp map[string]any
	endpoint := resourcePoolComponentsEndpoint(req.Target.OrgID, req.Target.ClusterID)
	if err := c.transport.getJSON(ctx, endpoint, nil, route.headers, route.trace, &resp); err != nil {
		return CloudClusterDetail{}, err
	}
	return decodeCloudClusterDetail(resp), nil
}
func (c *cloudClient) GetTopology(ctx context.Context, req CloudClusterTopologyRequest) (CloudClusterDetail, error) {
	target, err := req.Cluster.TopologyTarget()
	if err != nil {
		return CloudClusterDetail{}, err
	}
	detail, detailErr := c.GetClusterDetail(ctx, CloudClusterDetailRequest{
		Target:       target,
		TraceOrgType: req.Cluster.ClusterType,
	})
	if detailErr == nil && len(detail.Components) > 0 {
		return detail, nil
	}
	switch req.Cluster.NormalizedDeployType() {
	case "premium", "byoc", "resource_pool", "byoc_resource_pool", "premium_resource_pool":
		return c.GetResourcePoolComponents(ctx, CloudResourcePoolComponentsRequest{Target: target})
	default:
		if detailErr != nil {
			return CloudClusterDetail{}, detailErr
		}
		return detail, nil
	}
}
func decodeCloudClusterDetail(payload map[string]any) CloudClusterDetail {
	componentPayload := asAnyMap(payload["components"])
	topologyPayload := asAnyMap(payload["topology"])
	out := CloudClusterDetail{
		ID:           asTrimmedString(firstPresent(payload, "id", "clusterID", "clusterId")),
		Name:         asTrimmedString(firstPresent(payload, "name", "clusterName")),
		Components:   make(map[CloudClusterComponentType]CloudClusterComponent, len(componentPayload)+len(topologyPayload)),
		FeatureGates: decodeCloudClusterFeatureGates(asAnyMap(payload["featureGates"])),
	}
	for kind, component := range decodeCloudClusterComponents(componentPayload) {
		out.Components[kind] = component
	}
	for kind, replicas := range decodeCloudTopologySummary(topologyPayload) {
		component := out.Components[kind]
		if component.Replicas <= 0 {
			component.Replicas = replicas
		}
		out.Components[kind] = component
	}
	if len(out.Components) == 0 {
		out.Components = nil
	}
	return out
}
func decodeCloudClusterFeatureGates(payload map[string]any) CloudClusterFeatureGates {
	return CloudClusterFeatureGates{
		Known:                    len(payload) > 0,
		LogsEnabled:              asBoolOrFalse(firstPresent(payload, "logsEnabled", "logs_enabled")),
		SlowQueryEnabled:         asBoolOrFalse(firstPresent(payload, "slowQueryEnabled", "slow_query_enabled")),
		SlowQueryVisualEnabled:   asBoolOrFalse(firstPresent(payload, "slowQueryVisualEnabled", "slow_query_visual_enabled")),
		TopSQLEnabled:            asBoolOrFalse(firstPresent(payload, "topSQLEnabled", "top_sql_enabled")),
		ContinuousProfiling:      asBoolOrFalse(firstPresent(payload, "conProfEnabled", "continuousProfilingEnabled", "continuous_profiling_enabled")),
		BenchmarkReportEnabled:   asBoolOrFalse(firstPresent(payload, "benchmarkReportEnabled", "benchmark_report_enabled")),
		ComparisonReportEnabled:  asBoolOrFalse(firstPresent(payload, "comparisonReportEnabled", "comparison_report_enabled")),
		SystemCheckReportEnabled: asBoolOrFalse(firstPresent(payload, "systemCheckReportEnabled", "system_check_report_enabled")),
	}
}
func decodeCloudClusterComponents(payload map[string]any) map[CloudClusterComponentType]CloudClusterComponent {
	if len(payload) == 0 {
		return nil
	}
	out := make(map[CloudClusterComponentType]CloudClusterComponent, len(payload))
	for rawKind, rawComponent := range payload {
		kind, ok := normalizeCloudComponentType(rawKind)
		if !ok {
			continue
		}
		componentMap := asAnyMap(rawComponent)
		component := CloudClusterComponent{
			Replicas:            int(asInt64OrZero(firstPresent(componentMap, "replicas", "replica"))),
			TierName:            asTrimmedString(firstPresent(componentMap, "tierName", "tier_name")),
			StorageInstanceType: asTrimmedString(firstPresent(componentMap, "storageInstanceType", "storage_instance_type")),
		}
		storages := asAnyMap(componentMap["storages"])
		data := asAnyMap(firstPresent(storages, "data", "Data"))
		component.StorageIOPS = asInt64OrZero(firstPresent(data, "iops", "IOPS"))
		out[kind] = component
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
func decodeCloudTopologySummary(payload map[string]any) map[CloudClusterComponentType]int {
	if len(payload) == 0 {
		return nil
	}
	out := make(map[CloudClusterComponentType]int, len(payload))
	for rawKind, rawReplicas := range payload {
		kind, ok := normalizeCloudComponentType(rawKind)
		if !ok {
			continue
		}
		replicas := int(asInt64OrZero(rawReplicas))
		if replicas <= 0 {
			continue
		}
		out[kind] = replicas
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
func normalizeCloudComponentType(raw string) (CloudClusterComponentType, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "component_type_tidb", "tidb":
		return CloudClusterComponentTypeTiDB, true
	case "component_type_tikv", "tikv":
		return CloudClusterComponentTypeTiKV, true
	case "component_type_pd", "pd":
		return CloudClusterComponentTypePD, true
	case "component_type_tiflash", "tiflash", "ti_flash":
		return CloudClusterComponentTypeTiFlash, true
	default:
		return "", false
	}
}
func (c *cloudClient) QueryEvents(ctx context.Context, req CloudEventsRequest) (CloudEventsResult, error) {
	if c == nil || c.transport == nil {
		return CloudEventsResult{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromSharedControlPlaneTarget(cloudEventsPattern, req.Target, req.TraceOrgType)
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
func (c *cloudClient) GetEventDetail(ctx context.Context, req CloudEventDetailRequest) (map[string]any, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromSharedControlPlaneTarget(cloudEventsPattern, req.Target, req.TraceOrgType)
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
