package clinicapi

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AricSu/tidb-clinic-client/internal/model"
)

type (
	AuthProvider            = model.AuthProvider
	AuthProviderFunc        = model.AuthProviderFunc
	BearerTokenAuthProvider = model.BearerTokenAuthProvider
	ErrorClass              = model.ErrorClass
	Error                   = model.Error
	Hooks                   = model.Hooks
	RequestInfo             = model.RequestInfo
	RequestResult           = model.RequestResult
	RequestRetry            = model.RequestRetry
	RequestFailure          = model.RequestFailure
	Config                  = model.Config
	QueryMetadata           = model.QueryMetadata
	SeriesKind              = model.SeriesKind
	SeriesPoint             = model.SeriesPoint
	Series                  = model.Series
	SeriesResult            = model.SeriesResult
	MetricQueryRangeResult  = model.SeriesResult
	SlowQueryRecord         = model.SlowQueryRecord
	SlowQueryResult         = model.SlowQueryResult
	SlowQuerySamplesResult  = model.ListResult
)

const (
	ErrInvalidRequest = model.ErrInvalidRequest
	ErrUnsupported    = model.ErrUnsupported
	ErrAuth           = model.ErrAuth
	ErrNotFound       = model.ErrNotFound
	ErrNoData         = model.ErrNoData
	ErrTimeout        = model.ErrTimeout
	ErrRateLimit      = model.ErrRateLimit
	ErrDecode         = model.ErrDecode
	ErrBackend        = model.ErrBackend
	ErrTransient      = model.ErrTransient
)

var (
	StaticBearerToken = model.StaticBearerToken
	IsRetryable       = model.IsRetryable
	ClassOf           = model.ClassOf
	DefaultConfig     = model.DefaultConfig
)

type CatalogDataType int

const (
	CatalogDataTypeRetained          CatalogDataType = 1
	CatalogDataTypeLogs              CatalogDataType = 2
	CatalogDataTypeCollectedDownload CatalogDataType = 4
)

const (
	apiV1Prefix                = "/clinic/api/v1"
	ngmAPIV1Prefix             = "/ngm/api/v1"
	ngmContinuousProfilingBase = ngmAPIV1Prefix + "/continuous_profiling"
)

type RequestContext struct {
	OrgType        string
	RoutingOrgType string
	OrgID          string
	ClusterID      string
}

type ListClusterDataRequest struct {
	Context RequestContext
}

type EnsureCatalogDataReadableRequest struct {
	Context  RequestContext
	Item     ClinicDataItem
	DataType CatalogDataType
}

type CollectedDataDownloadRequest struct {
	Context RequestContext
	ItemID  string
}

type MetricsQueryRangeRequest struct {
	Context RequestContext
	Query   string
	Start   int64
	End     int64
	Step    string
	Timeout string
}

type SlowQueryRequest struct {
	Context   RequestContext
	ItemID    string
	StartTime int64
	EndTime   int64
	OrderBy   string
	Desc      bool
	Limit     int
}

type SlowQuerySamplesRequest struct {
	Context   RequestContext
	ItemID    string
	StartTime int64
	EndTime   int64
	Digest    string
	OrderBy   string
	Desc      bool
	Limit     int
	Fields    []string
}

type CloudTarget struct {
	OrgID     string
	ClusterID string
}

type CloudClusterLookupRequest struct {
	ClusterID   string
	ShowDeleted bool
}

type OrgRequest struct {
	OrgID string
}

type Org struct {
	ID   string
	Name string
	Type string
}

type CloudCluster struct {
	ClusterID    string
	Name         string
	ClusterType  string
	Provider     string
	Region       string
	DeployType   string
	DeployTypeV2 string
	ParentID     string
	OrgID        string
	TenantID     string
	ProjectID    string
	CreatedAt    int64
	DeletedAt    *int64
	Status       string
}

func (c CloudCluster) RequestContext() RequestContext {
	return RequestContext{
		OrgType:        strings.TrimSpace(c.ClusterType),
		RoutingOrgType: c.RoutingOrgType(),
		OrgID:          c.OrgID,
		ClusterID:      c.ClusterID,
	}
}

func (c CloudCluster) CloudTarget() CloudTarget {
	return CloudTarget{
		OrgID:     c.OrgID,
		ClusterID: c.ClusterID,
	}
}

func (c CloudCluster) CloudNGMTarget() CloudNGMTarget {
	return CloudNGMTarget{
		Provider:   c.Provider,
		Region:     c.Region,
		TenantID:   c.TenantID,
		ProjectID:  c.ProjectID,
		ClusterID:  c.ClusterID,
		DeployType: c.DeployType,
	}
}

func (c CloudCluster) NormalizedClusterType() string {
	return strings.ToLower(strings.TrimSpace(c.ClusterType))
}

func (c CloudCluster) NormalizedDeployType() string {
	switch value := strings.TrimSpace(c.DeployTypeV2); strings.ToLower(value) {
	case "":
		return strings.ToLower(strings.TrimSpace(c.DeployType))
	default:
		return strings.ToLower(value)
	}
}

func (c CloudCluster) RoutingOrgType() string {
	switch c.NormalizedClusterType() {
	case "cloud":
		return "cloud"
	case "tidb-cluster":
		return "op"
	}
	switch c.NormalizedDeployType() {
	case "dedicated", "shared", "starter", "essential", "premium", "byoc", "resource_pool", "premium_resource_pool", "byoc_resource_pool":
		return "cloud"
	case "tiup-cluster":
		return "op"
	}
	return "cloud"
}

func (c CloudCluster) HasParentCluster() bool {
	return strings.TrimSpace(c.ParentID) != ""
}

func (c CloudCluster) ResourcePoolTarget() (CloudTarget, error) {
	if strings.TrimSpace(c.OrgID) == "" {
		return CloudTarget{}, errors.New("org id is required")
	}
	if strings.TrimSpace(c.ParentID) == "" {
		return CloudTarget{}, errors.New("parent id is required")
	}
	return CloudTarget{OrgID: strings.TrimSpace(c.OrgID), ClusterID: strings.TrimSpace(c.ParentID)}, nil
}

func (c CloudCluster) ProfilingTarget() (CloudNGMTarget, error) {
	switch c.NormalizedDeployType() {
	case "premium", "byoc":
		if !c.HasParentCluster() {
			return CloudNGMTarget{}, errors.New("premium/byoc cluster requires parent id for profiling")
		}
		target := c.CloudNGMTarget()
		target.ClusterID = strings.TrimSpace(c.ParentID)
		return target, nil
	default:
		return c.CloudNGMTarget(), nil
	}
}

func (c CloudCluster) DiagnosticTarget() (CloudNGMTarget, error) {
	return c.ProfilingTarget()
}

func (c CloudCluster) MetricsParentFallbackLabel() (string, bool) {
	if !c.HasParentCluster() {
		return "", false
	}
	switch c.NormalizedDeployType() {
	case "premium", "byoc", "resource_pool", "premium_resource_pool", "byoc_resource_pool":
		return strings.TrimSpace(c.ParentID), true
	default:
		return "", false
	}
}

type CloudClusterDetailRequest struct {
	Target       CloudTarget
	TraceOrgType string
}

type CloudEventsRequest struct {
	Target       CloudTarget
	TraceOrgType string
	StartTime    int64
	EndTime      int64
	Name         string
	Severity     *int
	Limit        int
}

type CloudNGMTarget struct {
	Provider   string
	Region     string
	TenantID   string
	ProjectID  string
	ClusterID  string
	DeployType string
}

type ClinicDataItem struct {
	ItemID     string
	Filename   string
	Collectors []string
	HaveLog    bool
	HaveMetric bool
	HaveConfig bool
	StartTime  int64
	EndTime    int64
}

type CloudSlowQueryRequest struct {
	Target       CloudNGMTarget
	BeginTime    int64
	EndTime      int64
	OrderBy      string
	Desc         bool
	Digest       string
	Text         string
	Fields       []string
	Limit        int
	ShowInternal bool
}

type CloudSlowQueryDetailRequest struct {
	Target       CloudNGMTarget
	ConnectionID string
	Digest       string
	Timestamp    string
}

type CloudClusterFeatureGates struct {
	Known               bool
	LogsEnabled         bool
	ContinuousProfiling bool
}

type CloudClusterDetail struct {
	FeatureGates CloudClusterFeatureGates
}

type LokiQueryRangeRequest struct {
	ClusterID string
	Query     string
	Start     int64
	End       int64
	Limit     int
	Direction string
}

type LokiLabelsRequest struct {
	ClusterID string
	Start     int64
	End       int64
}

type LokiLabelValuesRequest struct {
	ClusterID string
	LabelName string
	Start     int64
	End       int64
}

type LokiLogValue struct {
	Timestamp string
	Line      string
}

type LokiStream struct {
	Labels map[string]string
	Values []LokiLogValue
}

type LokiQueryResult struct {
	Status     string
	ResultType string
	Streams    []LokiStream
	RawResult  []map[string]any
}

type LokiLabelsResult struct {
	Status string
	Values []string
}

type CloudDiagnosticListRequest struct {
	Target CloudNGMTarget
	Start  int64
	End    int64
}

type CloudDiagnosticFile struct {
	Name        string
	Key         string
	Size        int64
	DownloadURL string
	Raw         map[string]any
}

type CloudDiagnosticRecordGroup struct {
	Files []CloudDiagnosticFile
	Raw   map[string]any
}

type CloudDiagnosticListResult struct {
	Records []CloudDiagnosticRecordGroup
}

type CloudDiagnosticDownloadRequest struct {
	Target CloudNGMTarget
	Key    string
}

func defaultInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func asTrimmedString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func asFloat64OrZero(v any) float64 {
	if f, ok := asFloat64(v); ok {
		return f
	}
	return 0
}

func asInt64OrZero(v any) int64 {
	if n, ok := asInt64(v); ok {
		return n
	}
	return 0
}

func asBoolOrFalse(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "true")
	default:
		return false
	}
}

func asAnyMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func firstPresent(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return nil
}

func sliceOfMaps(v any) []map[string]any {
	switch x := v.(type) {
	case []map[string]any:
		return append([]map[string]any(nil), x...)
	case []any:
		out := make([]map[string]any, 0, len(x))
		for _, item := range x {
			if mapped, ok := item.(map[string]any); ok {
				out = append(out, mapped)
			}
		}
		return out
	default:
		return nil
	}
}

func sliceOfStrings(v any) []string {
	switch x := v.(type) {
	case []string:
		return append([]string(nil), x...)
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if value := asTrimmedString(item); value != "" {
				out = append(out, value)
			}
		}
		return out
	case string:
		value := strings.TrimSpace(x)
		if value == "" {
			return nil
		}
		return []string{value}
	default:
		return nil
	}
}

type clusterLookupItem struct {
	ClusterID    string `json:"clusterID"`
	Name         string `json:"clusterName"`
	ClusterType  string `json:"clusterType"`
	Provider     string `json:"clusterProviderName"`
	Region       string `json:"clusterRegionName"`
	DeployType   string `json:"clusterDeployType"`
	DeployTypeV2 string `json:"clusterDeployTypeV2"`
	ParentID     string `json:"parentID"`
	OrgID        string `json:"orgID"`
	TenantID     string `json:"tenantID"`
	ProjectID    string `json:"projectID"`
	CreatedAt    int64  `json:"clusterCreatedAt"`
	DeletedAt    int64  `json:"clusterDeletedAt"`
	Status       string `json:"clusterStatus"`
}

func decodeClusterLookupItem(item clusterLookupItem) CloudCluster {
	var deletedAt *int64
	if item.DeletedAt > 0 {
		value := item.DeletedAt
		deletedAt = &value
	}
	return CloudCluster{
		ClusterID:    strings.TrimSpace(item.ClusterID),
		Name:         strings.TrimSpace(item.Name),
		ClusterType:  strings.TrimSpace(item.ClusterType),
		Provider:     strings.TrimSpace(item.Provider),
		Region:       strings.TrimSpace(item.Region),
		DeployType:   strings.TrimSpace(item.DeployType),
		DeployTypeV2: strings.TrimSpace(item.DeployTypeV2),
		ParentID:     strings.TrimSpace(item.ParentID),
		OrgID:        strings.TrimSpace(item.OrgID),
		TenantID:     strings.TrimSpace(item.TenantID),
		ProjectID:    strings.TrimSpace(item.ProjectID),
		CreatedAt:    item.CreatedAt,
		DeletedAt:    deletedAt,
		Status:       strings.TrimSpace(item.Status),
	}
}
