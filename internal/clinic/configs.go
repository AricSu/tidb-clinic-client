package clinic

import (
	"context"
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

func (c *ConfigsClient) Get(ctx context.Context, query ConfigQuery) (ConfigResult, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return ConfigResult{}, &Error{Class: ErrBackend, Message: "configs client is nil"}
	}
	target, err := c.handle.requireCapability(ctx, CapabilityConfigs)
	if err != nil {
		return ConfigResult{}, err
	}
	requestContext, ok := target.requestContext()
	if !ok {
		return ConfigResult{}, &Error{Class: ErrBackend, Message: "collected-data request context is missing"}
	}
	if target.Platform != TargetPlatformTiUPCluster {
		return ConfigResult{}, unsupportedOperationError("capability:configs", "configs are only available for tiup-cluster collected data")
	}
	itemID, err := c.handle.client.resolveCatalogItemID(ctx, target, catalogIntentConfigs, 0, 0)
	if err != nil {
		return ConfigResult{}, err
	}
	return c.handle.client.clinic.GetConfig(ctx, requestContext, itemID, query)
}
func (c *clinicServiceClient) GetConfig(ctx context.Context, requestContext apitypes.RequestContext, itemID string, query ConfigQuery) (ConfigResult, error) {
	result, err := c.api.GetConfig(ctx, apitypes.ConfigRequest{
		Context: requestContext,
		ItemID:  itemID,
	})
	if err != nil {
		return ConfigResult{}, mapAPIError(err)
	}
	return tableResultFromConfig(result), nil
}
