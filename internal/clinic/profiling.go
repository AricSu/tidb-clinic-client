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
func (c *ProfilingClient) Detail(ctx context.Context, timestamp int64) (ProfileGroupDetail, error) {
	target, err := c.resolveProfilingTarget(ctx)
	if err != nil {
		return ProfileGroupDetail{}, err
	}
	result, err := c.handle.client.clinic.ProfileDetail(ctx, target, timestamp)
	if err != nil {
		return ProfileGroupDetail{}, err
	}
	return objectResultFromProfileDetail(result), nil
}
func (c *ProfilingClient) ActionToken(ctx context.Context, req ProfileActionTokenRequest) (string, error) {
	target, err := c.resolveProfilingTarget(ctx)
	if err != nil {
		return "", err
	}
	return c.handle.client.clinic.ProfileActionToken(ctx, target, req)
}
func (c *ProfilingClient) Download(ctx context.Context, req ProfileDownloadRequest) (DownloadedArtifact, error) {
	target, err := c.resolveProfilingTarget(ctx)
	if err != nil {
		return DownloadedArtifact{}, err
	}
	return c.handle.client.clinic.DownloadProfile(ctx, target, req)
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
	target, err := c.handle.resolveCloudTarget(ctx, CapabilityProfiling)
	if err != nil {
		return profilingTarget{}, err
	}
	return target.Profiling, nil
}
func (c *clinicServiceClient) ListProfileGroups(ctx context.Context, target profilingTarget, beginTime, endTime int64) (apitypes.CloudProfileGroupsResult, error) {
	result, err := c.api.ListProfileGroups(ctx, apitypes.CloudProfileGroupsRequest{
		Target:    target.cloudNGMTarget(),
		BeginTime: beginTime,
		EndTime:   endTime,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) ProfileDetail(ctx context.Context, target profilingTarget, timestamp int64) (apitypes.CloudProfileGroupDetail, error) {
	result, err := c.api.GetProfileGroupDetail(ctx, apitypes.CloudProfileGroupDetailRequest{
		Target:    target.cloudNGMTarget(),
		Timestamp: timestamp,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) ProfileActionToken(ctx context.Context, target profilingTarget, req ProfileActionTokenRequest) (string, error) {
	result, err := c.api.GetProfileActionToken(ctx, apitypes.CloudProfileActionTokenRequest{
		Target:      target.cloudNGMTarget(),
		Timestamp:   req.Timestamp,
		ProfileType: req.ProfileType,
		Component:   req.Component,
		Address:     req.Address,
		DataFormat:  req.DataFormat,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) DownloadProfile(ctx context.Context, target profilingTarget, req ProfileDownloadRequest) (DownloadedArtifact, error) {
	artifact, err := c.api.DownloadProfile(ctx, apitypes.CloudProfileDownloadRequest{
		Target: target.cloudNGMTarget(),
		Token:  req.Token,
	})
	if err != nil {
		return DownloadedArtifact{}, mapAPIError(err)
	}
	return downloadedArtifactFromCloud(artifact), nil
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
