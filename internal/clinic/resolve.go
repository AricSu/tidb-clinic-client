package clinic

import (
	"context"
	"fmt"
	clinicapi "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
	"strings"
)

const clustersResolveEndpoint = "clusters.resolve"

func (c *Client) resolveClusterBinding(ctx context.Context, selector ClusterSelector) (resolvedClusterTarget, error) {
	if c == nil || c.clinic == nil {
		return resolvedClusterTarget{}, &Error{Class: ErrBackend, Message: "clinic service is nil"}
	}
	selector = selector.normalized()
	if selector.ClusterID == "" {
		return resolvedClusterTarget{}, &Error{Class: ErrInvalidRequest, Endpoint: clustersResolveEndpoint, Message: "cluster id is required"}
	}
	cluster, err := c.clinic.ResolveCluster(ctx, selector)
	if err != nil {
		return resolvedClusterTarget{}, err
	}
	org, err := c.clinic.ResolveOrg(ctx, cluster.OrgID)
	if err != nil {
		return resolvedClusterTarget{}, err
	}
	platform := resolvedPlatformFromSharedMetadata(cluster, org)
	selector = resolvedIdentity(selector, platform, cluster.OrgID, cluster.ClusterID)
	return buildResolvedClusterTarget(selector, cluster), nil
}
func (c *Client) resolveTarget(ctx context.Context, selector ClusterSelector) (resolvedTarget, error) {
	if c == nil || c.clinic == nil {
		return resolvedTarget{}, &Error{Class: ErrBackend, Message: "clinic service is nil"}
	}
	selector = selector.normalized()
	if selector.ClusterID == "" {
		return resolvedTarget{}, &Error{Class: ErrInvalidRequest, Endpoint: clustersResolveEndpoint, Message: "cluster id is required"}
	}
	clusterBinding, err := c.resolveClusterBinding(ctx, selector)
	if err != nil {
		return resolvedTarget{}, err
	}
	if selector.Platform != "" && selector.Platform != clusterBinding.Identity.Platform {
		return resolvedTarget{}, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: clustersResolveEndpoint,
			Message:  fmt.Sprintf("cluster id %s resolved to platform %s, not %s", strings.TrimSpace(selector.ClusterID), clusterBinding.Identity.Platform, selector.Platform),
		}
	}
	switch selector.Platform {
	case "", TargetPlatformCloud, TargetPlatformTiUPCluster:
		return buildResolvedTargetFromCloud(clusterBinding), nil
	default:
		return resolvedTarget{}, &Error{Class: ErrInvalidRequest, Endpoint: clustersResolveEndpoint, Message: "unsupported target platform"}
	}
}
func resolvedIdentity(selector ClusterSelector, platform TargetPlatform, orgID, clusterID string) ClusterSelector {
	selector = selector.normalized()
	selector.Platform = platform
	selector.OrgID = firstNonEmpty(orgID, selector.OrgID)
	selector.ClusterID = firstNonEmpty(clusterID, selector.ClusterID)
	return selector
}
func resolvedPlatformFromSharedMetadata(cluster clinicapi.CloudCluster, org clinicapi.Org) TargetPlatform {
	switch cluster.NormalizedClusterType() {
	case "cloud":
		return TargetPlatformCloud
	case "tidb-cluster":
		return TargetPlatformTiUPCluster
	}
	orgType := strings.ToLower(strings.TrimSpace(org.Type))
	switch orgType {
	case "cloud":
		return TargetPlatformCloud
	case "op":
		return TargetPlatformTiUPCluster
	}
	switch cluster.NormalizedDeployType() {
	case "dedicated", "shared", "starter", "essential", "premium", "byoc", "resource_pool", "premium_resource_pool", "byoc_resource_pool":
		return TargetPlatformCloud
	default:
		if strings.TrimSpace(cluster.DeployType) == "" && strings.TrimSpace(cluster.DeployTypeV2) == "" {
			return TargetPlatformCloud
		}
		return TargetPlatformTiUPCluster
	}
}
func buildResolvedClusterTarget(selector ClusterSelector, cluster clinicapi.CloudCluster) resolvedClusterTarget {
	parentFallback, _ := cluster.MetricsParentFallbackLabel()
	logsClusterID := cluster.ClusterID
	orgType := cluster.RoutingOrgType()
	profilingNGMTarget := cluster.CloudNGMTarget()
	if next, err := cluster.ProfilingTarget(); err == nil {
		profilingNGMTarget = next
	}
	diagnosticsNGMTarget := cluster.CloudNGMTarget()
	if next, err := cluster.DiagnosticTarget(); err == nil {
		diagnosticsNGMTarget = next
	}
	return resolvedClusterTarget{
		Identity:     selector,
		ClusterID:    strings.TrimSpace(cluster.ClusterID),
		OrgID:        strings.TrimSpace(cluster.OrgID),
		ClusterType:  strings.TrimSpace(cluster.ClusterType),
		TenantID:     strings.TrimSpace(cluster.TenantID),
		ProjectID:    strings.TrimSpace(cluster.ProjectID),
		Provider:     strings.TrimSpace(cluster.Provider),
		Region:       strings.TrimSpace(cluster.Region),
		DeployType:   strings.TrimSpace(cluster.DeployType),
		DeployTypeV2: strings.TrimSpace(cluster.DeployTypeV2),
		ParentID:     strings.TrimSpace(cluster.ParentID),
		Status:       strings.TrimSpace(cluster.Status),
		Deleted:      cluster.DeletedAt != nil || strings.EqualFold(cluster.Status, "deleted"),
		ControlPlane: controlPlaneTarget{
			OrgID:        strings.TrimSpace(cluster.OrgID),
			ClusterID:    strings.TrimSpace(cluster.ClusterID),
			TraceOrgType: strings.TrimSpace(cluster.ClusterType),
		},
		Metrics: metricsTarget{
			Context: clinicapi.RequestContext{
				OrgType:        strings.TrimSpace(cluster.ClusterType),
				RoutingOrgType: orgType,
				OrgID:          strings.TrimSpace(cluster.OrgID),
				ClusterID:      strings.TrimSpace(cluster.ClusterID),
			},
			ParentFallbackLabel: strings.TrimSpace(parentFallback),
		},
		Logs: logsTarget{
			ClusterID: strings.TrimSpace(logsClusterID),
		},
		Profiling: profilingTarget{
			Provider:   profilingNGMTarget.Provider,
			Region:     profilingNGMTarget.Region,
			TenantID:   profilingNGMTarget.TenantID,
			ProjectID:  profilingNGMTarget.ProjectID,
			ClusterID:  profilingNGMTarget.ClusterID,
			DeployType: profilingNGMTarget.DeployType,
		},
		Diagnostics: diagnosticsTarget{
			Provider:   diagnosticsNGMTarget.Provider,
			Region:     diagnosticsNGMTarget.Region,
			TenantID:   diagnosticsNGMTarget.TenantID,
			ProjectID:  diagnosticsNGMTarget.ProjectID,
			ClusterID:  diagnosticsNGMTarget.ClusterID,
			DeployType: diagnosticsNGMTarget.DeployType,
		},
	}
}
func buildResolvedTargetFromCloud(target resolvedClusterTarget) resolvedTarget {
	platform := target.Identity.Platform
	if platform == "" {
		platform = TargetPlatformCloud
	}
	return resolvedTarget{
		Identity:     target.Identity,
		Platform:     platform,
		ClusterID:    target.ClusterID,
		OrgID:        target.OrgID,
		ClusterType:  target.ClusterType,
		Provider:     target.Provider,
		Region:       target.Region,
		DeployType:   target.DeployType,
		DeployTypeV2: target.DeployTypeV2,
		ParentID:     target.ParentID,
		Status:       target.Status,
		Deleted:      target.Deleted,
		Cloud:        &target,
	}
}
func unsupportedOperationError(endpoint, message string) error {
	return &Error{
		Class:    ErrUnsupported,
		Endpoint: endpoint,
		Message:  strings.TrimSpace(message),
	}
}
