package clinicapi

import (
	"github.com/AricSu/tidb-clinic-client/internal/clinic"
)

type (
	ClustersClient      = clinic.ClustersClient
	ClusterHandle       = clinic.ClusterHandle
	MetricsClient       = clinic.MetricsClient
	LogClient           = clinic.LogClient
	SlowQueryClient     = clinic.SlowQueryClient
	CollectedDataClient = clinic.CollectedDataClient
	ProfilingClient     = clinic.ProfilingClient
	DiagnosticsClient   = clinic.DiagnosticsClient
	TargetPlatform      = clinic.TargetPlatform
)

const (
	TargetPlatformCloud       = clinic.TargetPlatformCloud
	TargetPlatformTiUPCluster = clinic.TargetPlatformTiUPCluster
)
