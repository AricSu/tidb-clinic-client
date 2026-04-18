package clinic

import (
	"strings"

	clinicapi "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

type TargetPlatform string

const (
	TargetPlatformCloud       TargetPlatform = "cloud"
	TargetPlatformTiUPCluster TargetPlatform = "tiup-cluster"
)

type ClusterSelector struct {
	Platform  TargetPlatform
	OrgID     string
	ClusterID string
}

func (s ClusterSelector) normalized() ClusterSelector {
	s.Platform = TargetPlatform(strings.TrimSpace(string(s.Platform)))
	s.OrgID = strings.TrimSpace(s.OrgID)
	s.ClusterID = strings.TrimSpace(s.ClusterID)
	return s
}

type controlPlaneTarget struct {
	OrgID        string
	ClusterID    string
	TraceOrgType string
}

type metricsTarget struct {
	Context             clinicapi.RequestContext
	ParentFallbackLabel string
}

type logsTarget struct {
	ClusterID string
}

type profilingTarget struct {
	Provider   string
	Region     string
	TenantID   string
	ProjectID  string
	ClusterID  string
	DeployType string
}

type diagnosticsTarget struct {
	Provider   string
	Region     string
	TenantID   string
	ProjectID  string
	ClusterID  string
	DeployType string
}

type resolvedTiUPTarget struct {
	Context clinicapi.RequestContext
}

type resolvedClusterTarget struct {
	Identity     ClusterSelector
	ClusterID    string
	OrgID        string
	ClusterType  string
	TenantID     string
	ProjectID    string
	Provider     string
	Region       string
	DeployType   string
	DeployTypeV2 string
	ParentID     string
	Status       string
	Deleted      bool
	ControlPlane controlPlaneTarget
	Metrics      metricsTarget
	Logs         logsTarget
	Profiling    profilingTarget
	Diagnostics  diagnosticsTarget
}

type resolvedTarget struct {
	Identity     ClusterSelector
	Platform     TargetPlatform
	ClusterID    string
	OrgID        string
	ClusterType  string
	Provider     string
	Region       string
	DeployType   string
	DeployTypeV2 string
	ParentID     string
	Status       string
	Deleted      bool
	Cloud        *resolvedClusterTarget
	TiUP         *resolvedTiUPTarget
}

func (t controlPlaneTarget) cloudTarget() clinicapi.CloudTarget {
	return clinicapi.CloudTarget{
		OrgID:     strings.TrimSpace(t.OrgID),
		ClusterID: strings.TrimSpace(t.ClusterID),
	}
}

func (t profilingTarget) cloudNGMTarget() clinicapi.CloudNGMTarget {
	return clinicapi.CloudNGMTarget{
		Provider:   strings.TrimSpace(t.Provider),
		Region:     strings.TrimSpace(t.Region),
		TenantID:   strings.TrimSpace(t.TenantID),
		ProjectID:  strings.TrimSpace(t.ProjectID),
		ClusterID:  strings.TrimSpace(t.ClusterID),
		DeployType: strings.TrimSpace(t.DeployType),
	}
}

func (t diagnosticsTarget) cloudNGMTarget() clinicapi.CloudNGMTarget {
	return clinicapi.CloudNGMTarget{
		Provider:   strings.TrimSpace(t.Provider),
		Region:     strings.TrimSpace(t.Region),
		TenantID:   strings.TrimSpace(t.TenantID),
		ProjectID:  strings.TrimSpace(t.ProjectID),
		ClusterID:  strings.TrimSpace(t.ClusterID),
		DeployType: strings.TrimSpace(t.DeployType),
	}
}

func (t resolvedTarget) requestContext() (clinicapi.RequestContext, bool) {
	if t.Cloud != nil {
		return t.Cloud.Metrics.Context, true
	}
	switch t.Platform {
	case TargetPlatformTiUPCluster:
		if t.TiUP == nil {
			return clinicapi.RequestContext{}, false
		}
		return t.TiUP.Context, true
	default:
		return clinicapi.RequestContext{}, false
	}
}
