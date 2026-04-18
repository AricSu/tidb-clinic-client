package clinicapi

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

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

func clusterLookupQueryValues(queryText string, showDeleted bool, limit, page int) url.Values {
	query := url.Values{}
	query.Set("query", strings.TrimSpace(queryText))
	query.Set("show_deleted", strconv.FormatBool(showDeleted))
	query.Set("sort", "")
	query.Set("order", "")
	query.Set("limit", strconv.Itoa(defaultInt(limit, 10)))
	query.Set("page", strconv.Itoa(defaultInt(page, 1)))
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
	resp, err := c.getClusterDetail(ctx, req)
	if err != nil {
		return CloudClusterDetail{}, err
	}
	return decodeCloudClusterDetail(resp), nil
}

func (c *cloudClient) GetClusterDetailRaw(ctx context.Context, req CloudClusterDetailRequest) (map[string]any, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	resp, err := c.getClusterDetail(ctx, req)
	if err != nil {
		return nil, err
	}
	return cloneAnyMap(resp), nil
}

func (c *cloudClient) getClusterDetail(ctx context.Context, req CloudClusterDetailRequest) (map[string]any, error) {
	route, err := routeFromSharedControlPlaneTarget(clusterDetailPattern, req.Target, req.TraceOrgType)
	if err != nil {
		return nil, err
	}
	var resp map[string]any
	endpoint := clusterDetailEndpoint(req.Target.OrgID, req.Target.ClusterID)
	if err := c.transport.getJSON(ctx, endpoint, nil, route.headers, route.trace, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func decodeCloudClusterDetail(payload map[string]any) CloudClusterDetail {
	return CloudClusterDetail{
		FeatureGates: decodeCloudClusterFeatureGates(asAnyMap(payload["featureGates"])),
	}
}

func decodeCloudClusterFeatureGates(payload map[string]any) CloudClusterFeatureGates {
	return CloudClusterFeatureGates{
		Known:               len(payload) > 0,
		LogsEnabled:         asBoolOrFalse(firstPresent(payload, "logsEnabled", "logs_enabled")),
		ContinuousProfiling: asBoolOrFalse(firstPresent(payload, "conProfEnabled", "continuousProfilingEnabled", "continuous_profiling_enabled")),
	}
}

func (c *cloudClient) QueryEventsRaw(ctx context.Context, req CloudEventsRequest) (map[string]any, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromSharedControlPlaneTarget(cloudEventsPattern, req.Target, req.TraceOrgType)
	if err != nil {
		return nil, err
	}
	if req.StartTime <= 0 || req.EndTime <= 0 || req.EndTime < req.StartTime {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: cloudEventsPattern, Message: "valid start/end time range is required"}
	}
	query := url.Values{}
	query.Set("begin_ts", strconv.FormatInt(req.StartTime, 10))
	query.Set("end_ts", strconv.FormatInt(req.EndTime, 10))
	query.Set("name", strings.TrimSpace(req.Name))
	if req.Severity != nil {
		query.Set("severity", strconv.Itoa(*req.Severity))
	}
	var resp map[string]any
	endpoint := cloudEventsEndpoint(req.Target.OrgID, req.Target.ClusterID)
	if err := c.transport.getJSON(ctx, endpoint, query, route.headers, route.trace, &resp); err != nil {
		return nil, err
	}
	return cloneAnyMap(resp), nil
}
