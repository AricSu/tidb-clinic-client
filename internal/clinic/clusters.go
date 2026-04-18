package clinic

import (
	"context"
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

type ClusterHandle struct {
	client        *Client
	target        resolvedTarget
	Metrics       *MetricsClient
	Logs          *LogClient
	SlowQueries   *SlowQueryClient
	CollectedData *CollectedDataClient
	Profiling     *ProfilingClient
	Diagnostics   *DiagnosticsClient
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
	handle.SlowQueries = &SlowQueryClient{handle: handle}
	handle.CollectedData = &CollectedDataClient{handle: handle}
	handle.Profiling = &ProfilingClient{handle: handle}
	handle.Diagnostics = &DiagnosticsClient{handle: handle}
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
func (c *clinicServiceClient) ClusterDetail(ctx context.Context, target controlPlaneTarget) (apitypes.CloudClusterDetail, error) {
	result, err := c.api.GetClusterDetail(ctx, apitypes.CloudClusterDetailRequest{
		Target:       target.cloudTarget(),
		TraceOrgType: target.TraceOrgType,
	})
	return result, mapAPIError(err)
}
