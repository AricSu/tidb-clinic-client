package clinic

import (
	"context"
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
	"sync"
)

type ClusterHandle struct {
	client             *Client
	target             resolvedTarget
	capabilitiesMu     sync.Mutex
	capabilities       ClusterCapabilities
	capabilitiesLoaded bool
	Metrics            *MetricsClient
	Logs               *LogClient
	SQLAnalytics       *SQLAnalyticsClient
	Configs            *ConfigsClient
	Profiling          *ProfilingClient
	Diagnostics        *DiagnosticsClient
	Capabilities       *CapabilitiesClient
}

func newClusterHandle(client *Client, target resolvedTarget) *ClusterHandle {
	if client == nil {
		return nil
	}
	handle := &ClusterHandle{
		client: client,
		target: cloneResolvedTarget(target),
	}
	handle.Metrics = &MetricsClient{handle: handle}
	handle.Logs = &LogClient{handle: handle}
	handle.SQLAnalytics = &SQLAnalyticsClient{handle: handle}
	handle.Configs = &ConfigsClient{handle: handle}
	handle.Profiling = &ProfilingClient{handle: handle}
	handle.Diagnostics = &DiagnosticsClient{handle: handle}
	handle.Capabilities = &CapabilitiesClient{handle: handle}
	return handle
}
func (c *ClustersClient) Resolve(ctx context.Context, clusterID string) (*ClusterHandle, error) {
	return c.resolveSelector(ctx, ClusterSelector{ClusterID: clusterID})
}
func (c *ClustersClient) resolveSelector(ctx context.Context, selector ClusterSelector) (*ClusterHandle, error) {
	if c == nil || c.client == nil {
		return nil, &Error{Class: ErrBackend, Message: "clusters client is nil"}
	}
	target, err := c.client.resolveTarget(ctx, selector)
	if err != nil {
		return nil, err
	}
	return newClusterHandle(c.client, target), nil
}
func (c *ClustersClient) Search(ctx context.Context, query ClusterSearchQuery) ([]ClusterRecord, error) {
	if c == nil || c.client == nil || c.client.clinic == nil {
		return nil, &Error{Class: ErrBackend, Message: "clusters client is nil"}
	}
	items, err := c.client.clinic.SearchClusters(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make([]ClusterRecord, 0, len(items))
	for _, item := range items {
		out = append(out, clusterRecordFromCloud(item))
	}
	return out, nil
}
func (h *ClusterHandle) Platform() TargetPlatform {
	if h == nil {
		return ""
	}
	return h.target.Platform
}
func (h *ClusterHandle) ClusterID() string {
	if h == nil {
		return ""
	}
	return h.target.ClusterID
}
func (h *ClusterHandle) OrgID() string {
	if h == nil {
		return ""
	}
	return h.target.OrgID
}
func (h *ClusterHandle) Detail(ctx context.Context) (ClusterDetail, error) {
	target, err := h.resolveControlPlaneTarget(ctx, CapabilityClusterDetail)
	if err != nil {
		return ClusterDetail{}, err
	}
	detail, err := h.client.clinic.ClusterDetail(ctx, target.ControlPlane)
	if err != nil {
		return ClusterDetail{}, err
	}
	return clusterDetailFromCloud(target, detail), nil
}
func (h *ClusterHandle) Topology(ctx context.Context) (ClusterDetail, error) {
	target, err := h.resolveControlPlaneTarget(ctx, CapabilityTopology)
	if err != nil {
		return ClusterDetail{}, err
	}
	detail, err := h.client.clinic.Topology(ctx, target)
	if err != nil {
		return ClusterDetail{}, err
	}
	return clusterDetailFromCloud(target, detail), nil
}
func (h *ClusterHandle) Events(ctx context.Context, startTime, endTime int64) (ClusterEventsResult, error) {
	target, err := h.resolveControlPlaneTarget(ctx, CapabilityEvents)
	if err != nil {
		return ClusterEventsResult{}, err
	}
	result, err := h.client.clinic.Events(ctx, target.ControlPlane, startTime, endTime)
	if err != nil {
		return ClusterEventsResult{}, err
	}
	return clusterEventsFromCloud(result), nil
}
func (h *ClusterHandle) EventDetail(ctx context.Context, eventID string) (ClusterEventDetail, error) {
	target, err := h.resolveControlPlaneTarget(ctx, CapabilityEvents)
	if err != nil {
		return ClusterEventDetail{}, err
	}
	detail, err := h.client.clinic.EventDetail(ctx, target.ControlPlane, eventID)
	if err != nil {
		return ClusterEventDetail{}, err
	}
	return ClusterEventDetail{
		EventID: eventID,
		Detail:  cloneAnyMap(detail),
	}, nil
}
func (c *clinicServiceClient) ResolveCluster(ctx context.Context, selector ClusterSelector) (apitypes.CloudCluster, error) {
	result, err := c.api.GetCluster(ctx, apitypes.CloudClusterLookupRequest{
		ClusterID:   selector.ClusterID,
		ShowDeleted: true,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) ResolveOrg(ctx context.Context, orgID string) (apitypes.Org, error) {
	result, err := c.api.GetOrg(ctx, apitypes.OrgRequest{
		OrgID: orgID,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) SearchClusters(ctx context.Context, query ClusterSearchQuery) ([]apitypes.CloudCluster, error) {
	result, err := c.api.SearchClusters(ctx, apitypes.CloudClusterSearchRequest{
		Query:       query.Query,
		ClusterID:   query.ClusterID,
		ShowDeleted: query.ShowDeleted,
		Limit:       query.Limit,
		Page:        query.Page,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) ClusterDetail(ctx context.Context, target controlPlaneTarget) (apitypes.CloudClusterDetail, error) {
	result, err := c.api.GetClusterDetail(ctx, apitypes.CloudClusterDetailRequest{
		Target:       target.cloudTarget(),
		TraceOrgType: target.TraceOrgType,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) Topology(ctx context.Context, target resolvedClusterTarget) (apitypes.CloudClusterDetail, error) {
	result, err := c.api.GetTopology(ctx, apitypes.CloudClusterTopologyRequest{Cluster: target.cloudCluster()})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) Events(ctx context.Context, target controlPlaneTarget, startTime, endTime int64) (apitypes.CloudEventsResult, error) {
	result, err := c.api.QueryEvents(ctx, apitypes.CloudEventsRequest{
		Target:       target.cloudTarget(),
		TraceOrgType: target.TraceOrgType,
		StartTime:    startTime,
		EndTime:      endTime,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) EventDetail(ctx context.Context, target controlPlaneTarget, eventID string) (map[string]any, error) {
	result, err := c.api.GetEventDetail(ctx, apitypes.CloudEventDetailRequest{
		Target:       target.cloudTarget(),
		TraceOrgType: target.TraceOrgType,
		EventID:      eventID,
	})
	return result, mapAPIError(err)
}
