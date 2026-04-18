package clinic

import (
	"context"
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

func (c *ProfilingClient) ListGroups(ctx context.Context, beginTime, endTime int64) (ProfileGroupsResult, error) {
	target, err := c.resolveProfilingTarget(ctx)
	if err != nil {
		return ProfileGroupsResult{}, err
	}
	result, err := c.handle.client.clinic.ListProfileGroups(ctx, target, beginTime, endTime)
	if err != nil {
		return ProfileGroupsResult{}, err
	}
	return listResultFromProfileGroups(result), nil
}
func (c *ProfilingClient) Fetch(ctx context.Context, req ProfileFetchRequest) (DownloadedArtifact, error) {
	target, err := c.resolveProfilingTarget(ctx)
	if err != nil {
		return DownloadedArtifact{}, err
	}
	return c.handle.client.clinic.FetchProfile(ctx, target, req)
}
func (c *ProfilingClient) resolveProfilingTarget(ctx context.Context) (profilingTarget, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return profilingTarget{}, &Error{Class: ErrBackend, Message: "profiling client is nil"}
	}
	target, err := c.handle.requireTarget("profiling client")
	if err != nil {
		return profilingTarget{}, err
	}
	if target.Deleted {
		return profilingTarget{}, unsupportedOperationError("profiling", "profiling is unavailable for deleted clusters")
	}
	if target.Platform != TargetPlatformCloud || target.Cloud == nil {
		return profilingTarget{}, unsupportedOperationError("profiling", "profiling is unavailable for tiup-cluster collected data")
	}
	if target.isSharedTier() {
		return profilingTarget{}, unsupportedOperationError("profiling", "profiling is unavailable for shared/starter/essential clusters")
	}
	gates, err := c.handle.loadClusterFeatureGates(ctx, "profiling client")
	if err != nil {
		return profilingTarget{}, err
	}
	if gates.Known && !gates.ContinuousProfiling {
		return profilingTarget{}, unsupportedOperationError("profiling", "profiling is disabled by cluster featureGates")
	}
	return target.Cloud.Profiling, nil
}
func (c *clinicServiceClient) ListProfileGroups(ctx context.Context, target profilingTarget, beginTime, endTime int64) (apitypes.CloudProfileGroupsResult, error) {
	result, err := c.api.ListProfileGroups(ctx, apitypes.CloudProfileGroupsRequest{
		Target:    target.cloudNGMTarget(),
		BeginTime: beginTime,
		EndTime:   endTime,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) FetchProfile(ctx context.Context, target profilingTarget, req ProfileFetchRequest) (DownloadedArtifact, error) {
	artifact, err := c.api.FetchProfile(ctx, apitypes.CloudProfileFetchRequest{
		Target:      target.cloudNGMTarget(),
		Timestamp:   req.Timestamp,
		ProfileType: req.ProfileType,
		Component:   req.Component,
		Address:     req.Address,
		DataFormat:  req.DataFormat,
	})
	if err != nil {
		return DownloadedArtifact{}, mapAPIError(err)
	}
	return downloadedArtifactFromCloud(artifact), nil
}
