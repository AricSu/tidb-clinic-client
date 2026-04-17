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
func (c *Client) discoverCapabilitiesForTarget(ctx context.Context, target resolvedTarget) (ClusterCapabilities, error) {
	if c == nil || c.clinic == nil {
		return ClusterCapabilities{}, &Error{Class: ErrBackend, Message: "clinic service is nil"}
	}
	capabilities := staticCapabilityDescriptors(target, declaredCapabilities())
	overrides := realCapabilityOverrides(ctx, c.clinic, target)
	for i, descriptor := range capabilities {
		if override, ok := overrides[descriptor.Name]; ok {
			if override.Name == "" {
				override.Name = descriptor.Name
			}
			if override.Scope == "" {
				override.Scope = descriptor.Scope
			}
			if override.Stability == "" {
				override.Stability = descriptor.Stability
			}
			if len(override.TierConstraints) == 0 {
				override.TierConstraints = append([]string(nil), descriptor.TierConstraints...)
			} else {
				override.TierConstraints = append([]string(nil), override.TierConstraints...)
			}
			capabilities[i] = override
		}
	}
	return ClusterCapabilities{
		Cluster:      clusterMetadataFromResolvedTarget(target),
		Capabilities: capabilities,
	}, nil
}
func buildResolvedClusterTarget(selector ClusterSelector, cluster clinicapi.CloudCluster) resolvedClusterTarget {
	parentFallback, _ := cluster.MetricsParentFallbackLabel()
	logsClusterID := cluster.ClusterID
	sqlClusterID := cluster.ClusterID
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
		SQL: sqlTarget{
			ClusterID: strings.TrimSpace(sqlClusterID),
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
func declaredCapabilities() map[CapabilityName]bool {
	return map[CapabilityName]bool{
		CapabilityClusterDetail:   true,
		CapabilityTopology:        true,
		CapabilityEvents:          true,
		CapabilityMetrics:         true,
		CapabilityLogs:            true,
		CapabilitySQLQuery:        true,
		CapabilitySchema:          true,
		CapabilityTopSQL:          true,
		CapabilitySlowQuery:       true,
		CapabilitySQLStatements:   true,
		CapabilityConfigs:         true,
		CapabilityProfiling:       true,
		CapabilityDiagnosticFiles: true,
	}
}
func staticCapabilityDescriptors(target resolvedTarget, defaultSupport map[CapabilityName]bool) []CapabilityDescriptor {
	deployType := strings.ToLower(strings.TrimSpace(target.DeployTypeV2))
	if deployType == "" {
		deployType = strings.ToLower(strings.TrimSpace(target.DeployType))
	}
	isDeleted := target.Deleted
	isSharedTier := deployType == "shared" || deployType == "starter" || deployType == "essential"
	requiresParent := strings.TrimSpace(target.ParentID) != "" && (deployType == "premium" || deployType == "byoc")
	profilingTiers := []string{"dedicated", "premium", "byoc", "resource_pool", "premium_resource_pool", "byoc_resource_pool"}
	topologyTiers := append([]string(nil), profilingTiers...)
	diagnosticTiers := append([]string(nil), profilingTiers...)
	dataPlaneReason := "data-plane capability is unavailable for deleted clusters"
	liveReason := "capability requires a live cluster target"
	topologyReason := "topology is unavailable for deleted or shared/starter/essential clusters"
	configsReason := "configs are only available for tiup-cluster collected data"
	isCloud := target.Platform == TargetPlatformCloud
	descriptor := func(name CapabilityName, available bool, reason string, requiresParentTarget bool, requiresLiveCluster bool, tierConstraints []string) CapabilityDescriptor {
		if !defaultSupport[name] {
			available = false
			if reason == "" {
				reason = "capability is not implemented by the current Clinic API service"
			}
		}
		return CapabilityDescriptor{
			Name:                 name,
			Available:            available,
			Reason:               reason,
			Scope:                CapabilityScopeCluster,
			Stability:            CapabilityStabilityStable,
			RequiresParentTarget: requiresParentTarget,
			RequiresLiveCluster:  requiresLiveCluster,
			TierConstraints:      append([]string(nil), tierConstraints...),
		}
	}
	capabilities := []CapabilityDescriptor{
		descriptor(CapabilityClusterDetail, true, "", false, false, nil),
		descriptor(CapabilityTopology, !isDeleted && !isSharedTier, reasonIf(isDeleted || isSharedTier, topologyReason), requiresParent, true, topologyTiers),
		descriptor(CapabilityEvents, !isDeleted, reasonIf(isDeleted, liveReason), false, true, nil),
		descriptor(CapabilityMetrics, !isDeleted, reasonIf(isDeleted, dataPlaneReason), false, true, nil),
		descriptor(CapabilityLogs, isCloud && !isDeleted, reasonIf(!isCloud || isDeleted, dataPlaneReasonForPlatform(isCloud, isDeleted, "logs are only available for cloud clusters", "logs are only available from tiup-cluster collected data")), false, true, nil),
		descriptor(CapabilitySQLQuery, isCloud && !isDeleted, reasonIf(!isCloud || isDeleted, dataPlaneReasonForPlatform(isCloud, isDeleted, "sql query is only available for cloud clusters", "sql query is unavailable for tiup-cluster collected data")), false, true, nil),
		descriptor(CapabilitySchema, isCloud && !isDeleted, reasonIf(!isCloud || isDeleted, dataPlaneReasonForPlatform(isCloud, isDeleted, "schema is only available for cloud clusters", "schema is unavailable for tiup-cluster collected data")), false, true, nil),
		descriptor(CapabilityTopSQL, isCloud && !isDeleted, reasonIf(!isCloud || isDeleted, dataPlaneReasonForPlatform(isCloud, isDeleted, "topsql is only available for cloud clusters", "topsql is unavailable for tiup-cluster collected data")), false, true, nil),
		descriptor(CapabilitySlowQuery, isCloud && !isDeleted, reasonIf(!isCloud || isDeleted, dataPlaneReasonForPlatform(isCloud, isDeleted, "slow query is only available for cloud clusters", "slow query records are only available from tiup-cluster collected data")), false, true, nil),
		descriptor(CapabilitySQLStatements, isCloud && !isDeleted, reasonIf(!isCloud || isDeleted, dataPlaneReasonForPlatform(isCloud, isDeleted, "sql statements are only available for cloud clusters", "sql statements are unavailable for tiup-cluster collected data")), false, true, nil),
		descriptor(CapabilityConfigs, false, configsReason, false, false, nil),
		descriptor(CapabilityProfiling, isCloud && !isDeleted && !isSharedTier, reasonIf(!isCloud || isDeleted || isSharedTier, profilingReasonForPlatform(isCloud, isDeleted, isSharedTier)), requiresParent, true, profilingTiers),
		descriptor(CapabilityDiagnosticFiles, isCloud && !isDeleted && !isSharedTier, reasonIf(!isCloud || isDeleted || isSharedTier, diagnosticReasonForPlatform(isCloud, isDeleted, isSharedTier)), requiresParent, true, diagnosticTiers),
	}
	return capabilities
}
func realCapabilityOverrides(ctx context.Context, clinic clinicService, target resolvedTarget) map[CapabilityName]CapabilityDescriptor {
	overrides := map[CapabilityName]CapabilityDescriptor{}
	switch target.Platform {
	case TargetPlatformCloud:
		if target.Cloud == nil {
			return nil
		}
		detail, err := clinic.ClusterDetail(ctx, target.Cloud.ControlPlane)
		if err != nil {
			return nil
		}
		mergeCapabilityOverrides(overrides, featureGateCapabilityOverrides(detail.FeatureGates))
	case TargetPlatformTiUPCluster:
		requestContext, ok := target.requestContext()
		if !ok {
			return nil
		}
		items, err := clinic.ListCatalogData(ctx, requestContext)
		if err != nil {
			return nil
		}
		mergeCapabilityOverrides(overrides, catalogCapabilityOverrides(items))
	}
	if len(overrides) == 0 {
		return nil
	}
	return overrides
}
func mergeCapabilityOverrides(dst map[CapabilityName]CapabilityDescriptor, src map[CapabilityName]CapabilityDescriptor) {
	for name, descriptor := range src {
		descriptor.Name = name
		dst[name] = descriptor
	}
}
func featureGateCapabilityOverrides(gates clinicapi.CloudClusterFeatureGates) map[CapabilityName]CapabilityDescriptor {
	if !gates.Known {
		return nil
	}
	overrides := map[CapabilityName]CapabilityDescriptor{}
	if !gates.LogsEnabled {
		overrides[CapabilityLogs] = CapabilityDescriptor{
			Available:           false,
			Reason:              "logs are disabled by cluster featureGates",
			RequiresLiveCluster: true,
		}
	}
	if !gates.TopSQLEnabled {
		overrides[CapabilityTopSQL] = CapabilityDescriptor{
			Available:           false,
			Reason:              "topsql is disabled by cluster featureGates",
			RequiresLiveCluster: true,
		}
	}
	if !gates.AllowsSlowQuery() {
		overrides[CapabilitySlowQuery] = CapabilityDescriptor{
			Available:           false,
			Reason:              "slow query is disabled by cluster featureGates",
			RequiresLiveCluster: true,
		}
	}
	if !gates.ContinuousProfiling {
		overrides[CapabilityProfiling] = CapabilityDescriptor{
			Available:           false,
			Reason:              "profiling is disabled by cluster featureGates",
			RequiresLiveCluster: true,
		}
	}
	return overrides
}
func catalogCapabilityOverrides(items []clinicapi.ClinicDataItem) map[CapabilityName]CapabilityDescriptor {
	hasLogs := false
	hasConfigs := false
	hasSlowQuery := false
	for _, item := range items {
		hasLogs = hasLogs || item.HaveLog
		hasConfigs = hasConfigs || item.HaveConfig
		hasSlowQuery = hasSlowQuery || (item.HaveLog && hasCollector(item.Collectors, "log.slow"))
	}
	return map[CapabilityName]CapabilityDescriptor{
		CapabilityLogs: {
			Available:           hasLogs,
			Reason:              reasonIf(!hasLogs, "logs require collected data with logs"),
			RequiresLiveCluster: false,
		},
		CapabilitySlowQuery: {
			Available:           hasSlowQuery,
			Reason:              reasonIf(!hasSlowQuery, "slow query requires collected data with log.slow"),
			RequiresLiveCluster: false,
		},
		CapabilityConfigs: {
			Available:           hasConfigs,
			Reason:              reasonIf(!hasConfigs, "configs require collected data with config snapshots"),
			RequiresLiveCluster: false,
		},
	}
}
func dataPlaneReasonForPlatform(isCloud, isDeleted bool, cloudReason, tiupReason string) string {
	if isDeleted {
		return "data-plane capability is unavailable for deleted clusters"
	}
	if isCloud {
		return cloudReason
	}
	return tiupReason
}
func profilingReasonForPlatform(isCloud, isDeleted, isSharedTier bool) string {
	if !isCloud {
		return "profiling is unavailable for tiup-cluster collected data"
	}
	if isDeleted {
		return "profiling is unavailable for deleted clusters"
	}
	if isSharedTier {
		return "profiling is unavailable for shared/starter/essential clusters"
	}
	return ""
}
func diagnosticReasonForPlatform(isCloud, isDeleted, isSharedTier bool) string {
	if !isCloud {
		return "diagnostic files are unavailable for tiup-cluster clusters"
	}
	if isDeleted {
		return "diagnostic files are unavailable for deleted clusters"
	}
	if isSharedTier {
		return "diagnostic files are unavailable for shared/starter/essential clusters"
	}
	return ""
}
func requireCapability(capabilities ClusterCapabilities, name CapabilityName, clusterID string) error {
	descriptor, ok := capabilities.Lookup(name)
	if !ok {
		return &Error{
			Class:    ErrUnsupported,
			Endpoint: "capability:" + string(name),
			Message:  fmt.Sprintf("capability %s is unknown for cluster %s", name, strings.TrimSpace(clusterID)),
		}
	}
	if descriptor.Available {
		return nil
	}
	reason := strings.TrimSpace(descriptor.Reason)
	if reason == "" {
		reason = fmt.Sprintf("capability %s is unavailable for cluster %s", name, strings.TrimSpace(clusterID))
	}
	return &Error{
		Class:    ErrUnsupported,
		Endpoint: "capability:" + string(name),
		Message:  reason,
	}
}
func reasonIf(condition bool, reason string) string {
	if condition {
		return reason
	}
	return ""
}
func unsupportedOperationError(endpoint, message string) error {
	return &Error{
		Class:    ErrUnsupported,
		Endpoint: endpoint,
		Message:  strings.TrimSpace(message),
	}
}
