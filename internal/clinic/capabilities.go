package clinic

import (
	"context"
	"fmt"
)

func (h *ClusterHandle) discoverCapabilities(ctx context.Context) (ClusterCapabilities, error) {
	if h == nil || h.client.clinic == nil {
		return ClusterCapabilities{}, &Error{Class: ErrBackend, Message: "cluster handle is nil"}
	}
	h.capabilitiesMu.Lock()
	if h.capabilitiesLoaded {
		cached := cloneClusterCapabilities(h.capabilities)
		h.capabilitiesMu.Unlock()
		return cached, nil
	}
	h.capabilitiesMu.Unlock()
	discovered, err := h.client.discoverCapabilitiesForTarget(ctx, h.target)
	if err != nil {
		return ClusterCapabilities{}, err
	}
	h.capabilitiesMu.Lock()
	if !h.capabilitiesLoaded {
		h.capabilities = cloneClusterCapabilities(discovered)
		h.capabilitiesLoaded = true
	}
	cached := cloneClusterCapabilities(h.capabilities)
	h.capabilitiesMu.Unlock()
	return cached, nil
}
func (h *ClusterHandle) requireCapability(ctx context.Context, capability CapabilityName) (resolvedTarget, error) {
	if h == nil || h.client.clinic == nil {
		return resolvedTarget{}, &Error{Class: ErrBackend, Message: "cluster handle is nil"}
	}
	capabilities, err := h.discoverCapabilities(ctx)
	if err != nil {
		return resolvedTarget{}, err
	}
	if err := requireCapability(capabilities, capability, h.target.ClusterID); err != nil {
		return resolvedTarget{}, err
	}
	return cloneResolvedTarget(h.target), nil
}
func (h *ClusterHandle) resolveCloudTarget(ctx context.Context, capability CapabilityName) (resolvedClusterTarget, error) {
	target, err := h.requireCapability(ctx, capability)
	if err != nil {
		return resolvedClusterTarget{}, err
	}
	if target.Platform != TargetPlatformCloud || target.Cloud == nil {
		return resolvedClusterTarget{}, unsupportedOperationError("clusters", fmt.Sprintf("capability %s is only available for cloud targets", capability))
	}
	return *target.Cloud, nil
}
func (h *ClusterHandle) resolveControlPlaneTarget(ctx context.Context, capability CapabilityName) (resolvedClusterTarget, error) {
	target, err := h.requireCapability(ctx, capability)
	if err != nil {
		return resolvedClusterTarget{}, err
	}
	if target.Cloud == nil {
		return resolvedClusterTarget{}, unsupportedOperationError("clusters", fmt.Sprintf("capability %s is unavailable for the resolved cluster", capability))
	}
	return *target.Cloud, nil
}
func (c *CapabilitiesClient) Discover(ctx context.Context) (ClusterCapabilities, error) {
	if c == nil || c.handle == nil {
		return ClusterCapabilities{}, &Error{Class: ErrBackend, Message: "capabilities client is nil"}
	}
	return c.handle.discoverCapabilities(ctx)
}
