package clinic

import (
	"context"
	"strings"

	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

func (c *CollectedDataClient) List(ctx context.Context) ([]CollectedDataItem, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return nil, &Error{Class: ErrBackend, Message: "collected data client is nil"}
	}
	target := cloneResolvedTarget(c.handle.target)
	requestContext, ok := target.requestContext()
	if !ok {
		return nil, &Error{Class: ErrBackend, Message: "collected-data request context is missing"}
	}
	if target.Platform != TargetPlatformTiUPCluster {
		return nil, unsupportedOperationError("collected_data", "collected data is only available for non-cloud / OP deployments; cloud clusters are not supported")
	}
	items, err := c.handle.client.clinic.ListCatalogData(ctx, requestContext)
	if err != nil {
		return nil, err
	}
	out := make([]CollectedDataItem, 0, len(items))
	for _, item := range items {
		out = append(out, collectedDataItemFromAPI(item))
	}
	return out, nil
}

func (c *CollectedDataClient) Download(ctx context.Context, req CollectedDataDownloadRequest) (DownloadedArtifact, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return DownloadedArtifact{}, &Error{Class: ErrBackend, Message: "collected data client is nil"}
	}
	target := cloneResolvedTarget(c.handle.target)
	requestContext, ok := target.requestContext()
	if !ok {
		return DownloadedArtifact{}, &Error{Class: ErrBackend, Message: "collected-data request context is missing"}
	}
	if target.Platform != TargetPlatformTiUPCluster {
		return DownloadedArtifact{}, unsupportedOperationError("collected_data.download", "collected data download is only available for non-cloud / OP deployments; cloud clusters are not supported")
	}
	items, err := c.handle.client.clinic.ListCatalogData(ctx, requestContext)
	if err != nil {
		return DownloadedArtifact{}, err
	}
	item, err := selectCatalogItem(catalogIntentCollectedData, items, req.StartTime, req.EndTime)
	if err != nil {
		return DownloadedArtifact{}, err
	}
	return c.handle.client.clinic.DownloadCollectedData(ctx, requestContext, item)
}

func (c *clinicServiceClient) DownloadCollectedData(ctx context.Context, requestContext apitypes.RequestContext, item apitypes.ClinicDataItem) (DownloadedArtifact, error) {
	if err := c.EnsureCatalogDataReadable(ctx, requestContext, item, apitypes.CatalogDataTypeCollectedDownload); err != nil {
		return DownloadedArtifact{}, err
	}
	body, err := c.api.DownloadCollectedData(ctx, apitypes.CollectedDataDownloadRequest{
		Context: requestContext,
		ItemID:  item.ItemID,
	})
	if err != nil {
		return DownloadedArtifact{}, mapAPIError(err)
	}
	return DownloadedArtifact{
		Filename: firstNonEmpty(item.Filename, item.ItemID),
		Bytes:    body,
	}, nil
}

func collectedDataItemFromAPI(item apitypes.ClinicDataItem) CollectedDataItem {
	return CollectedDataItem{
		ItemID:     strings.TrimSpace(item.ItemID),
		Filename:   strings.TrimSpace(item.Filename),
		Collectors: append([]string(nil), item.Collectors...),
		HaveLog:    item.HaveLog,
		HaveMetric: item.HaveMetric,
		HaveConfig: item.HaveConfig,
		StartTime:  item.StartTime,
		EndTime:    item.EndTime,
	}
}
