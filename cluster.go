package clinicapi

import (
	"github.com/AricSu/tidb-clinic-client/internal/clinic"
	"github.com/AricSu/tidb-clinic-client/internal/model"
)

type (
	ClustersClient       = clinic.ClustersClient
	ClusterHandle        = clinic.ClusterHandle
	MetricsClient        = clinic.MetricsClient
	LogClient            = clinic.LogClient
	SQLAnalyticsClient   = clinic.SQLAnalyticsClient
	ConfigsClient        = clinic.ConfigsClient
	ProfilingClient      = clinic.ProfilingClient
	DiagnosticsClient    = clinic.DiagnosticsClient
	CapabilitiesClient   = clinic.CapabilitiesClient
	TargetPlatform       = clinic.TargetPlatform
	ClusterRecord        = clinic.ClusterRecord
	ClusterMetadata      = clinic.ClusterMetadata
	ClusterSearchQuery   = clinic.ClusterSearchQuery
	ClusterComponentType = clinic.ClusterComponentType
	ClusterComponent     = clinic.ClusterComponent
	ClusterDetail        = clinic.ClusterDetail
	ClusterEvent         = clinic.ClusterEvent
	ClusterEventsResult  = clinic.ClusterEventsResult
	ClusterEventDetail   = clinic.ClusterEventDetail
	CapabilityName       = model.CapabilityName
	CapabilityScope      = model.CapabilityScope
	CapabilityStability  = model.CapabilityStability
	CapabilityDescriptor = model.CapabilityDescriptor
	ClusterCapabilities  = model.ClusterCapabilities
)

const (
	TargetPlatformCloud            = clinic.TargetPlatformCloud
	TargetPlatformTiUPCluster      = clinic.TargetPlatformTiUPCluster
	ClusterComponentTypeTiDB       = clinic.ClusterComponentTypeTiDB
	ClusterComponentTypeTiKV       = clinic.ClusterComponentTypeTiKV
	ClusterComponentTypePD         = clinic.ClusterComponentTypePD
	ClusterComponentTypeTiFlash    = clinic.ClusterComponentTypeTiFlash
	CapabilityClusterDetail        = model.CapabilityClusterDetail
	CapabilityTopology             = model.CapabilityTopology
	CapabilityEvents               = model.CapabilityEvents
	CapabilityMetrics              = model.CapabilityMetrics
	CapabilityLogs                 = model.CapabilityLogs
	CapabilitySQLQuery             = model.CapabilitySQLQuery
	CapabilitySchema               = model.CapabilitySchema
	CapabilityTopSQL               = model.CapabilityTopSQL
	CapabilitySlowQuery            = model.CapabilitySlowQuery
	CapabilitySQLStatements        = model.CapabilitySQLStatements
	CapabilityConfigs              = model.CapabilityConfigs
	CapabilityProfiling            = model.CapabilityProfiling
	CapabilityDiagnosticFiles      = model.CapabilityDiagnosticFiles
	CapabilityScopeCluster         = model.CapabilityScopeCluster
	CapabilityStabilityStable      = model.CapabilityStabilityStable
	CapabilityStabilityPlaceholder = model.CapabilityStabilityPlaceholder
)
