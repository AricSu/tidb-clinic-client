package clinic

import (
	"context"
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

func (c *DiagnosticsClient) ListPlanReplayer(ctx context.Context, start, end int64) (DiagnosticListResult, error) {
	target, err := c.resolveDiagnosticsTarget(ctx)
	if err != nil {
		return DiagnosticListResult{}, err
	}
	result, err := c.handle.client.clinic.ListPlanReplayer(ctx, target, start, end)
	if err != nil {
		return DiagnosticListResult{}, err
	}
	return listResultFromDiagnostics(result), nil
}
func (c *DiagnosticsClient) ListOOMRecord(ctx context.Context, start, end int64) (DiagnosticListResult, error) {
	target, err := c.resolveDiagnosticsTarget(ctx)
	if err != nil {
		return DiagnosticListResult{}, err
	}
	result, err := c.handle.client.clinic.ListOOMRecord(ctx, target, start, end)
	if err != nil {
		return DiagnosticListResult{}, err
	}
	return listResultFromDiagnostics(result), nil
}
func (c *DiagnosticsClient) Download(ctx context.Context, req DiagnosticDownloadRequest) (DownloadedArtifact, error) {
	target, err := c.resolveDiagnosticsTarget(ctx)
	if err != nil {
		return DownloadedArtifact{}, err
	}
	return c.handle.client.clinic.DownloadDiagnostic(ctx, target, req)
}
func (c *DiagnosticsClient) resolveDiagnosticsTarget(ctx context.Context) (diagnosticsTarget, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return diagnosticsTarget{}, &Error{Class: ErrBackend, Message: "diagnostics client is nil"}
	}
	target, err := c.handle.resolveCloudTarget(ctx, CapabilityDiagnosticFiles)
	if err != nil {
		return diagnosticsTarget{}, err
	}
	return target.Diagnostics, nil
}
func (c *clinicServiceClient) ListPlanReplayer(ctx context.Context, target diagnosticsTarget, start, end int64) (apitypes.CloudDiagnosticListResult, error) {
	result, err := c.api.ListPlanReplayerArtifacts(ctx, apitypes.CloudDiagnosticListRequest{
		Target: target.cloudNGMTarget(),
		Start:  start,
		End:    end,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) ListOOMRecord(ctx context.Context, target diagnosticsTarget, start, end int64) (apitypes.CloudDiagnosticListResult, error) {
	result, err := c.api.ListOOMRecordArtifacts(ctx, apitypes.CloudDiagnosticListRequest{
		Target: target.cloudNGMTarget(),
		Start:  start,
		End:    end,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) DownloadDiagnostic(ctx context.Context, target diagnosticsTarget, req DiagnosticDownloadRequest) (DownloadedArtifact, error) {
	artifact, err := c.api.DownloadDiagnosticArtifact(ctx, apitypes.CloudDiagnosticDownloadRequest{
		Target: target.cloudNGMTarget(),
		Key:    req.Key,
	})
	if err != nil {
		return DownloadedArtifact{}, mapAPIError(err)
	}
	return downloadedArtifactFromCloud(artifact), nil
}
