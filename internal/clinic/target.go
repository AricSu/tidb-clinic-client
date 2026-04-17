package clinic

import (
	"fmt"
	clinicapi "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
	"strings"
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
type sqlTarget struct {
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
	SQL          sqlTarget
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
type ClusterRecord struct {
	ClusterID    string
	Name         string
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
}
type ClusterMetadata struct {
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
}
type ClusterSearchQuery struct {
	Query       string
	ClusterID   string
	ShowDeleted bool
	Limit       int
	Page        int
}
type ClusterComponentType string

const (
	ClusterComponentTypeTiDB    ClusterComponentType = "COMPONENT_TYPE_TIDB"
	ClusterComponentTypeTiKV    ClusterComponentType = "COMPONENT_TYPE_TIKV"
	ClusterComponentTypePD      ClusterComponentType = "COMPONENT_TYPE_PD"
	ClusterComponentTypeTiFlash ClusterComponentType = "COMPONENT_TYPE_TIFLASH"
)

type ClusterComponent struct {
	Replicas            int
	TierName            string
	StorageInstanceType string
	StorageIOPS         int64
}
type ClusterDetail struct {
	Cluster    ClusterRecord
	ID         string
	Name       string
	Components map[ClusterComponentType]ClusterComponent
}

func (d ClusterDetail) Topology() string {
	parts := make([]string, 0, 4)
	appendPart := func(kind ClusterComponentType, short string) {
		component, ok := d.Components[kind]
		if !ok {
			return
		}
		part := fmt.Sprintf("%d-%s", component.Replicas, short)
		if strings.TrimSpace(component.TierName) != "" {
			part += "-" + strings.TrimSpace(component.TierName)
		}
		if component.StorageInstanceType != "" {
			if component.StorageIOPS > 0 {
				part += fmt.Sprintf(" (%s, IOPS: %d)", component.StorageInstanceType, component.StorageIOPS)
			} else {
				part += fmt.Sprintf(" (%s)", component.StorageInstanceType)
			}
		}
		parts = append(parts, part)
	}
	appendPart(ClusterComponentTypeTiDB, "tidb")
	appendPart(ClusterComponentTypeTiKV, "tikv")
	appendPart(ClusterComponentTypePD, "pd")
	appendPart(ClusterComponentTypeTiFlash, "tiflash")
	return strings.Join(parts, " / ")
}

type ClusterEvent struct {
	EventID     string
	Name        string
	DisplayName string
	CreateTime  int64
	Payload     map[string]any
}
type ClusterEventsResult struct {
	Total  int
	Events []ClusterEvent
}
type ClusterEventDetail struct {
	EventID string
	Detail  map[string]any
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
func clusterRecordFromCloud(cluster clinicapi.CloudCluster) ClusterRecord {
	return ClusterRecord{
		ClusterID:    strings.TrimSpace(cluster.ClusterID),
		Name:         strings.TrimSpace(cluster.Name),
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
	}
}
func clusterDetailFromCloud(target resolvedClusterTarget, detail clinicapi.CloudClusterDetail) ClusterDetail {
	components := make(map[ClusterComponentType]ClusterComponent, len(detail.Components))
	for kind, component := range detail.Components {
		mappedKind, ok := clusterComponentTypeFromCloud(kind)
		if !ok {
			continue
		}
		components[mappedKind] = ClusterComponent{
			Replicas:            component.Replicas,
			TierName:            strings.TrimSpace(component.TierName),
			StorageInstanceType: strings.TrimSpace(component.StorageInstanceType),
			StorageIOPS:         component.StorageIOPS,
		}
	}
	record := ClusterRecord{
		ClusterID:    strings.TrimSpace(target.ClusterID),
		Name:         strings.TrimSpace(detail.Name),
		OrgID:        strings.TrimSpace(target.OrgID),
		ClusterType:  strings.TrimSpace(target.ClusterType),
		TenantID:     strings.TrimSpace(target.TenantID),
		ProjectID:    strings.TrimSpace(target.ProjectID),
		Provider:     strings.TrimSpace(target.Provider),
		Region:       strings.TrimSpace(target.Region),
		DeployType:   strings.TrimSpace(target.DeployType),
		DeployTypeV2: strings.TrimSpace(target.DeployTypeV2),
		ParentID:     strings.TrimSpace(target.ParentID),
		Status:       strings.TrimSpace(target.Status),
		Deleted:      target.Deleted,
	}
	return ClusterDetail{
		Cluster:    record,
		ID:         strings.TrimSpace(detail.ID),
		Name:       strings.TrimSpace(detail.Name),
		Components: components,
	}
}
func clusterEventsFromCloud(result clinicapi.CloudEventsResult) ClusterEventsResult {
	events := make([]ClusterEvent, 0, len(result.Events))
	for _, event := range result.Events {
		events = append(events, ClusterEvent{
			EventID:     strings.TrimSpace(event.EventID),
			Name:        strings.TrimSpace(event.Name),
			DisplayName: strings.TrimSpace(event.DisplayName),
			CreateTime:  event.CreateTime,
			Payload:     cloneAnyMap(event.Payload),
		})
	}
	return ClusterEventsResult{
		Total:  result.Total,
		Events: events,
	}
}
func clusterComponentTypeFromCloud(kind clinicapi.CloudClusterComponentType) (ClusterComponentType, bool) {
	switch kind {
	case clinicapi.CloudClusterComponentTypeTiDB:
		return ClusterComponentTypeTiDB, true
	case clinicapi.CloudClusterComponentTypeTiKV:
		return ClusterComponentTypeTiKV, true
	case clinicapi.CloudClusterComponentTypePD:
		return ClusterComponentTypePD, true
	case clinicapi.CloudClusterComponentTypeTiFlash:
		return ClusterComponentTypeTiFlash, true
	default:
		return "", false
	}
}
func (t resolvedClusterTarget) cloudCluster() clinicapi.CloudCluster {
	return clinicapi.CloudCluster{
		ClusterID:    strings.TrimSpace(t.ClusterID),
		OrgID:        strings.TrimSpace(t.OrgID),
		ClusterType:  strings.TrimSpace(t.ClusterType),
		TenantID:     strings.TrimSpace(t.TenantID),
		ProjectID:    strings.TrimSpace(t.ProjectID),
		Provider:     strings.TrimSpace(t.Provider),
		Region:       strings.TrimSpace(t.Region),
		DeployType:   strings.TrimSpace(t.DeployType),
		DeployTypeV2: strings.TrimSpace(t.DeployTypeV2),
		ParentID:     strings.TrimSpace(t.ParentID),
		Status:       strings.TrimSpace(t.Status),
	}
}
func (t resolvedClusterTarget) topSQLTarget() clinicapi.CloudNGMTarget {
	return clinicapi.CloudNGMTarget{
		Provider:   strings.TrimSpace(t.Provider),
		Region:     strings.TrimSpace(t.Region),
		TenantID:   strings.TrimSpace(t.TenantID),
		ProjectID:  strings.TrimSpace(t.ProjectID),
		ClusterID:  strings.TrimSpace(t.ClusterID),
		DeployType: strings.TrimSpace(t.DeployType),
	}
}
