package clinicapi

import (
	"errors"
	"fmt"
	"github.com/AricSu/tidb-clinic-client/internal/model"
	"net/url"
	"strings"
)

type (
	AuthProvider             = model.AuthProvider
	AuthProviderFunc         = model.AuthProviderFunc
	BearerTokenAuthProvider  = model.BearerTokenAuthProvider
	ErrorClass               = model.ErrorClass
	Error                    = model.Error
	Hooks                    = model.Hooks
	RequestInfo              = model.RequestInfo
	RequestResult            = model.RequestResult
	RequestRetry             = model.RequestRetry
	RequestFailure           = model.RequestFailure
	Config                   = model.Config
	QueryMetadata            = model.QueryMetadata
	SeriesKind               = model.SeriesKind
	SeriesPoint              = model.SeriesPoint
	Series                   = model.Series
	SeriesResult             = model.SeriesResult
	MetricQueryRangeResult   = model.SeriesResult
	MetricQueryInstantResult = model.SeriesResult
	MetricQuerySeriesResult  = model.SeriesResult
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

const (
	apiV1Prefix                = "/clinic/api/v1"
	ngmAPIV1Prefix             = "/ngm/api/v1"
	resourcePoolPattern        = apiV1Prefix + "/orgs/{orgID}/clusters/{clusterID}/resource-pool/components"
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
type MetricsQueryRangeRequest struct {
	Context RequestContext
	Query   string
	Start   int64
	End     int64
	Step    string
	Timeout string
}
type MetricsQueryInstantRequest struct {
	Context RequestContext
	Query   string
	Time    int64
	Timeout string
}
type MetricsQuerySeriesRequest struct {
	Context RequestContext
	Match   []string
	Start   int64
	End     int64
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
type CollectedSlowQueryListRequest struct {
	Context   RequestContext
	ItemID    string
	StartTime int64
	EndTime   int64
	Digest    string
	OrderBy   string
	Desc      bool
	Limit     int
}
type CollectedSlowQueryDetailRequest struct {
	Context     RequestContext
	ItemID      string
	SlowQueryID string
}
type LogSearchRequest struct {
	Context   RequestContext
	ItemID    string
	StartTime int64
	EndTime   int64
	Pattern   string
	Limit     int
}
type ConfigRequest struct {
	Context RequestContext
	ItemID  string
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
func (c CloudCluster) TopologyTarget() (CloudTarget, error) {
	switch c.NormalizedDeployType() {
	case "dedicated", "resource_pool", "byoc_resource_pool", "premium_resource_pool":
		return c.CloudTarget(), nil
	case "premium", "byoc":
		return c.ResourcePoolTarget()
	case "shared", "starter", "essential":
		return CloudTarget{}, errors.New("shared/starter/essential clusters do not expose topology")
	default:
		if c.HasParentCluster() {
			return c.ResourcePoolTarget()
		}
		return c.CloudTarget(), nil
	}
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
}
type CloudEventDetailRequest struct {
	Target       CloudTarget
	TraceOrgType string
	EventID      string
}
type CloudNGMTarget struct {
	Provider   string
	Region     string
	TenantID   string
	ProjectID  string
	ClusterID  string
	DeployType string
}
type CloudTopSQLSummaryRequest struct {
	Target    CloudNGMTarget
	Component string
	Instance  string
	Start     string
	End       string
	Top       int
	Window    string
	GroupBy   string
}
type CloudTopSlowQueriesRequest struct {
	Target  CloudNGMTarget
	Start   string
	Hours   int
	OrderBy string
	Limit   int
}
type CloudSlowQueryListRequest struct {
	Target  CloudNGMTarget
	Digest  string
	Start   string
	End     string
	OrderBy string
	Limit   int
	Desc    bool
	Fields  []string
}
type CloudSlowQueryDetailRequest struct {
	Target       CloudNGMTarget
	Digest       string
	ConnectionID string
	Timestamp    string
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

func resourcePoolComponentsEndpoint(orgID, clusterID string) string {
	return fmt.Sprintf(
		"%s/orgs/%s/clusters/%s/resource-pool/components",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
	)
}

type CloudClusterSearchRequest struct {
	Query       string
	ClusterID   string
	ShowDeleted bool
	Limit       int
	Page        int
}
type CloudClusterTopologyRequest struct {
	Cluster CloudCluster
}
type CloudResourcePoolComponentsRequest struct {
	Target CloudTarget
}
type CloudClusterComponentType string

const (
	CloudClusterComponentTypeTiDB    CloudClusterComponentType = "COMPONENT_TYPE_TIDB"
	CloudClusterComponentTypeTiKV    CloudClusterComponentType = "COMPONENT_TYPE_TIKV"
	CloudClusterComponentTypePD      CloudClusterComponentType = "COMPONENT_TYPE_PD"
	CloudClusterComponentTypeTiFlash CloudClusterComponentType = "COMPONENT_TYPE_TIFLASH"
)

type CloudClusterComponent struct {
	Replicas            int
	TierName            string
	StorageInstanceType string
	StorageIOPS         int64
}
type CloudClusterFeatureGates struct {
	Known                    bool
	LogsEnabled              bool
	SlowQueryEnabled         bool
	SlowQueryVisualEnabled   bool
	TopSQLEnabled            bool
	ContinuousProfiling      bool
	BenchmarkReportEnabled   bool
	ComparisonReportEnabled  bool
	SystemCheckReportEnabled bool
}

func (g CloudClusterFeatureGates) AllowsSlowQuery() bool {
	return g.SlowQueryEnabled || g.SlowQueryVisualEnabled
}

type CloudClusterDetail struct {
	ID           string
	Name         string
	Components   map[CloudClusterComponentType]CloudClusterComponent
	FeatureGates CloudClusterFeatureGates
}

func (d CloudClusterDetail) Topology() string {
	parts := make([]string, 0, 4)
	appendPart := func(kind CloudClusterComponentType, short string) {
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
	appendPart(CloudClusterComponentTypeTiDB, "tidb")
	appendPart(CloudClusterComponentTypeTiKV, "tikv")
	appendPart(CloudClusterComponentTypePD, "pd")
	appendPart(CloudClusterComponentTypeTiFlash, "tiflash")
	return strings.Join(parts, " / ")
}

type CloudClusterEvent struct {
	EventID     string
	Name        string
	DisplayName string
	CreateTime  int64
	Payload     map[string]any
}
type CloudEventsResult struct {
	Total  int
	Events []CloudClusterEvent
}
type CloudTopSQLPlan struct {
	PlanDigest        string
	PlanText          string
	TimestampSec      []int64
	CPUTimeMS         []int64
	ExecCountPerSec   float64
	DurationPerExecMS float64
	ScanRecordsPerSec float64
	ScanIndexesPerSec float64
}
type CloudTopSQL struct {
	SQLDigest         string
	SQLText           string
	CPUTimeMS         float64
	ExecCountPerSec   float64
	DurationPerExecMS float64
	ScanRecordsPerSec float64
	ScanIndexesPerSec float64
	Plans             []CloudTopSQLPlan
}
type CloudTopSlowQuery struct {
	DB            string
	SQLDigest     string
	SQLText       string
	StatementType string
	Count         int64
	SumLatency    float64
	MaxLatency    float64
	AvgLatency    float64
	SumMemory     float64
	MaxMemory     float64
	AvgMemory     float64
	SumDisk       float64
	MaxDisk       float64
	AvgDisk       float64
	Detail        map[string]any
}
type CloudSlowQueryListEntry struct {
	ID           string
	ItemID       string
	Digest       string
	Query        string
	Timestamp    string
	QueryTime    float64
	MemoryMax    float64
	RequestCount int64
	ConnectionID string
	Raw          map[string]any
}
type DataProxyQueryRequest struct {
	ClusterID string
	SQL       string
	Timeout   int
}
type DataProxyQueryResult struct {
	Columns  []string
	Rows     [][]string
	Metadata QueryMetadata
}
type DataProxySchemaRequest struct {
	ClusterID string
	Tables    []string
}
type DataProxySchemaColumn struct {
	Name    string
	Type    string
	Comment string
}
type DataProxySchemaPartition struct {
	Name string
	Type string
}
type DataProxyTableSchema struct {
	Database   string
	Table      string
	Columns    []DataProxySchemaColumn
	Partitions []DataProxySchemaPartition
	DataSource string
	Location   string
	Raw        map[string]any
}
type DataProxySchemaResult struct {
	Tables []DataProxyTableSchema
}
type LokiQueryRequest struct {
	ClusterID string
	Query     string
	Time      int64
	Limit     int
	Direction string
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
type SlowQueryRecord struct {
	Digest     string
	SQLText    string
	QueryTime  float64
	ExecCount  int64
	User       string
	DB         string
	TableNames []string
	IndexNames []string
	SourceRef  string
}
type SlowQueryRecordsResult struct {
	Total   int
	Records []SlowQueryRecord
}
type LogRecord struct {
	Timestamp int64
	Component string
	Level     string
	Message   string
	SourceRef string
}
type LogSearchResult struct {
	Total   int
	Records []LogRecord
}
type ConfigEntry struct {
	Component string
	Key       string
	Value     string
	SourceRef string
}
type ConfigResult struct {
	Entries []ConfigEntry
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
func defaultString(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}
func defaultFields(fields []string) []string {
	if len(fields) == 0 {
		return []string{"query", "timestamp", "query_time", "memory_max", "request_count", "digest", "connection_id"}
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{"query", "timestamp", "query_time", "memory_max", "request_count", "digest", "connection_id"}
	}
	return out
}
func ngmInstance(component, instance, clusterID string) string {
	port := "10080"
	if strings.EqualFold(strings.TrimSpace(component), "tikv") {
		port = "20160"
	}
	return strings.TrimSpace(instance) + ".db-" + strings.TrimSpace(component) + "-peer.tidb" + strings.TrimSpace(clusterID) + ".svc:" + port
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
func sliceOfStrings(v any) []string {
	switch x := v.(type) {
	case []string:
		return append([]string(nil), x...)
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			value := asTrimmedString(item)
			if value != "" {
				out = append(out, value)
			}
		}
		return out
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
	Topology     any    `json:"topology"`
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
