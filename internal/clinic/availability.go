package clinic

import (
	"context"
	"strings"

	clinicapi "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

func (h *ClusterHandle) requireTarget(clientName string) (resolvedTarget, error) {
	if h == nil || h.client == nil || h.client.clinic == nil {
		return resolvedTarget{}, &Error{Class: ErrBackend, Message: clientName + " is nil"}
	}
	return cloneResolvedTarget(h.target), nil
}

func (h *ClusterHandle) requireClusterTarget(clientName string) (resolvedClusterTarget, error) {
	target, err := h.requireTarget(clientName)
	if err != nil {
		return resolvedClusterTarget{}, err
	}
	if target.Cloud == nil {
		return resolvedClusterTarget{}, &Error{Class: ErrBackend, Message: "resolved cluster target is missing"}
	}
	return *target.Cloud, nil
}

func (h *ClusterHandle) loadClusterFeatureGates(ctx context.Context, clientName string) (clinicapi.CloudClusterFeatureGates, error) {
	target, err := h.requireClusterTarget(clientName)
	if err != nil {
		return clinicapi.CloudClusterFeatureGates{}, err
	}
	detail, err := h.client.clinic.ClusterDetail(ctx, target.ControlPlane)
	if err != nil {
		return clinicapi.CloudClusterFeatureGates{}, err
	}
	return detail.FeatureGates, nil
}

func (h *ClusterHandle) loadCatalogItems(ctx context.Context, clientName string) ([]clinicapi.ClinicDataItem, resolvedTarget, error) {
	target, err := h.requireTarget(clientName)
	if err != nil {
		return nil, resolvedTarget{}, err
	}
	requestContext, ok := target.requestContext()
	if !ok {
		return nil, resolvedTarget{}, &Error{Class: ErrBackend, Message: "collected-data request context is missing"}
	}
	items, err := h.client.clinic.ListCatalogData(ctx, requestContext)
	if err != nil {
		return nil, resolvedTarget{}, err
	}
	return items, target, nil
}

func (target resolvedTarget) isSharedTier() bool {
	deployType := target.normalizedDeployType()
	return deployType == "shared" || deployType == "starter" || deployType == "essential"
}

func (target resolvedTarget) normalizedDeployType() string {
	if target.DeployTypeV2 != "" {
		return normalizeLower(target.DeployTypeV2)
	}
	return normalizeLower(target.DeployType)
}

func (target resolvedClusterTarget) isSharedTier() bool {
	deployType := target.normalizedDeployType()
	return deployType == "shared" || deployType == "starter" || deployType == "essential"
}

func (target resolvedClusterTarget) normalizedDeployType() string {
	if target.DeployTypeV2 != "" {
		return normalizeLower(target.DeployTypeV2)
	}
	return normalizeLower(target.DeployType)
}

func normalizeLower(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func catalogHasLogs(items []clinicapi.ClinicDataItem) bool {
	for _, item := range items {
		if item.HaveLog {
			return true
		}
	}
	return false
}
