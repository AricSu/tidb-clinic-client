package clinicapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const apiV1Prefix = "/clinic/api/v1"
const ngmAPIV1Prefix = "/ngm/api/v1"

const (
	cloudClusterLookupPath = apiV1Prefix + "/dashboard/clusters2"
	metricsEndpoint        = apiV1Prefix + "/data/metrics"
	slowQueriesEndpoint    = apiV1Prefix + "/data/slowqueries"
	logsEndpoint           = apiV1Prefix + "/data/logs"
	configEndpoint         = apiV1Prefix + "/data/config"
	catalogEndpointPattern = apiV1Prefix + "/orgs/{orgID}/clusters/{clusterID}/data"
	clusterDetailPattern   = apiV1Prefix + "/orgs/{orgID}/clusters/{clusterID}"
	cloudEventsPattern     = apiV1Prefix + "/activityhub/applications/{orgID}/targets/{clusterID}/activities"
	ngmTopSQLEndpoint      = ngmAPIV1Prefix + "/topsql/summary"
	ngmTopSlowQueriesPath  = ngmAPIV1Prefix + "/slow_query/stats"
	ngmSlowQueryListPath   = ngmAPIV1Prefix + "/slow_query/list"
	ngmSlowQueryDetailPath = ngmAPIV1Prefix + "/slow_query/detail"
)

const maxSamplesErrorMessage = "cannot select more than -search.maxSamplesPerQuery"

// RequestContext identifies the Clinic tenant and cluster for a request.
type RequestContext struct {
	OrgType   string
	OrgID     string
	ClusterID string
}

// Hooks defines optional request lifecycle callbacks.
type Hooks struct {
	OnRequestStart func(RequestInfo)
	OnRequestDone  func(RequestResult)
	OnRetry        func(RequestRetry)
	OnError        func(RequestFailure)
}

// RequestInfo describes a single request attempt before completion.
type RequestInfo struct {
	Endpoint  string
	Method    string
	Attempt   int
	OrgType   string
	OrgID     string
	ClusterID string
	ItemID    string
}

// RequestResult describes a completed request attempt.
type RequestResult struct {
	RequestInfo
	StatusCode    int
	Duration      time.Duration
	ResponseBytes int
}

// RequestRetry describes a retryable request failure.
type RequestRetry struct {
	RequestResult
	ErrorClass ErrorClass
	Retryable  bool
	Err        error
}

// RequestFailure describes a terminal request failure.
type RequestFailure struct {
	RequestResult
	ErrorClass ErrorClass
	Retryable  bool
	Err        error
}

type requestRoute struct {
	headers http.Header
	trace   requestTrace
}

type knownClusterRouteInput struct {
	orgType   string
	orgID     string
	clusterID string
	itemID    string
	headers   http.Header
}

// ListClusterDataRequest requests the uploaded data catalog for a cluster.
type ListClusterDataRequest struct {
	Context RequestContext
}

// MetricsQueryRangeRequest requests a time-ranged metrics query.
type MetricsQueryRangeRequest struct {
	Context RequestContext
	Query   string
	Start   int64
	End     int64
	Step    string
}

// SlowQueryRequest requests slow query data for an uploaded item and time
// range.
type SlowQueryRequest struct {
	Context   RequestContext
	ItemID    string
	StartTime int64
	EndTime   int64
	OrderBy   string
	Desc      bool
	Limit     int
}

// LogSearchRequest requests log search results for an uploaded item.
type LogSearchRequest struct {
	Context   RequestContext
	ItemID    string
	StartTime int64
	EndTime   int64
	Pattern   string
	Limit     int
}

// ConfigRequest requests a config snapshot for an uploaded item.
type ConfigRequest struct {
	Context RequestContext
	ItemID  string
}

// CloudTarget identifies a known cloud org and cluster.
type CloudTarget struct {
	OrgID     string
	ClusterID string
}

// CloudClusterLookupRequest requests cloud routing metadata for a known cluster.
type CloudClusterLookupRequest struct {
	ClusterID   string
	ShowDeleted bool
}

// CloudCluster describes one cloud cluster lookup result.
type CloudCluster struct {
	ClusterID  string
	Name       string
	Provider   string
	Region     string
	DeployType string
	OrgID      string
	TenantID   string
	ProjectID  string
	CreatedAt  int64
	DeletedAt  *int64
	Status     string
}

// RequestContext converts the cluster metadata into a shared cloud request context.
func (c CloudCluster) RequestContext() RequestContext {
	return RequestContext{
		OrgType:   "cloud",
		OrgID:     c.OrgID,
		ClusterID: c.ClusterID,
	}
}

// CloudTarget converts the cluster metadata into a cloud control-plane target.
func (c CloudCluster) CloudTarget() CloudTarget {
	return CloudTarget{
		OrgID:     c.OrgID,
		ClusterID: c.ClusterID,
	}
}

// CloudNGMTarget converts the cluster metadata into a cloud NGM routing target.
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

// CloudClusterDetailRequest requests cluster detail for a known cloud target.
type CloudClusterDetailRequest struct {
	Target CloudTarget
}

// CloudEventsRequest requests cluster activity events for a known cloud target.
type CloudEventsRequest struct {
	Target    CloudTarget
	StartTime int64
	EndTime   int64
}

// CloudEventDetailRequest requests a specific cloud event detail object.
type CloudEventDetailRequest struct {
	Target  CloudTarget
	EventID string
}

// CloudNGMTarget identifies a known cloud cluster together with the extra NGM
// routing metadata required by cloud-only diagnostics endpoints.
type CloudNGMTarget struct {
	Provider   string
	Region     string
	TenantID   string
	ProjectID  string
	ClusterID  string
	DeployType string
}

// CloudTopSQLSummaryRequest requests an NGM TopSQL summary for a known cloud
// target and component instance.
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

// CloudTopSlowQueriesRequest requests the slow-query aggregate summary from
// cloud NGM for a known target.
type CloudTopSlowQueriesRequest struct {
	Target  CloudNGMTarget
	Start   string
	Hours   int
	OrderBy string
	Limit   int
}

// CloudSlowQueryListRequest requests slow-query samples for one digest from
// cloud NGM.
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

// CloudSlowQueryDetailRequest requests one detailed slow-query record from
// cloud NGM.
type CloudSlowQueryDetailRequest struct {
	Target       CloudNGMTarget
	Digest       string
	ConnectionID string
	Timestamp    string
}

// ClinicDataItem describes one uploaded diagnostic bundle known to Clinic.
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

// MetricSample represents one metric sample point.
type MetricSample struct {
	Timestamp int64
	Value     string
}

// MetricSeries represents one labeled metric series.
type MetricSeries struct {
	Labels map[string]string
	Values []MetricSample
}

// MetricQueryRangeResult is the typed result of a metrics range query.
type MetricQueryRangeResult struct {
	IsPartial  bool
	ResultType string
	Series     []MetricSeries
}

// SlowQueryRecord describes one slow query entry.
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

// SlowQueryResult is the typed result of a slow query request.
type SlowQueryResult struct {
	Total   int
	Records []SlowQueryRecord
}

// LogRecord describes one log search hit.
type LogRecord struct {
	Timestamp int64
	Component string
	Level     string
	Message   string
	SourceRef string
}

// LogSearchResult is the typed result of a log search request.
type LogSearchResult struct {
	Total   int
	Records []LogRecord
}

// ConfigEntry describes one config key-value pair.
type ConfigEntry struct {
	Component string
	Key       string
	Value     string
	SourceRef string
}

// ConfigSnapshot is the typed result of a config fetch request.
type ConfigSnapshot struct {
	Entries []ConfigEntry
}

// CloudClusterComponentType identifies a component in a cloud cluster detail
// response.
type CloudClusterComponentType string

const (
	CloudClusterComponentTypeTiDB    CloudClusterComponentType = "COMPONENT_TYPE_TIDB"
	CloudClusterComponentTypeTiKV    CloudClusterComponentType = "COMPONENT_TYPE_TIKV"
	CloudClusterComponentTypePD      CloudClusterComponentType = "COMPONENT_TYPE_PD"
	CloudClusterComponentTypeTiFlash CloudClusterComponentType = "COMPONENT_TYPE_TIFLASH"
)

// CloudClusterComponent describes a component entry in a cloud cluster detail
// response.
type CloudClusterComponent struct {
	Replicas            int
	TierName            string
	StorageInstanceType string
	StorageIOPS         int64
}

// CloudClusterDetail describes a known cloud cluster.
type CloudClusterDetail struct {
	ID         string
	Name       string
	Components map[CloudClusterComponentType]CloudClusterComponent
}

// Topology returns a compact topology string derived from the cluster
// components when available.
func (d CloudClusterDetail) Topology() string {
	parts := make([]string, 0, 4)
	appendPart := func(kind CloudClusterComponentType, short string) {
		component, ok := d.Components[kind]
		if !ok {
			return
		}
		part := fmt.Sprintf("%d-%s-%s", component.Replicas, short, component.TierName)
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

// CloudClusterEvent describes a single cloud activity event.
type CloudClusterEvent struct {
	EventID     string
	Name        string
	DisplayName string
	CreateTime  int64
	Payload     map[string]any
}

// CloudEventsResult is the typed result of a cloud event query.
type CloudEventsResult struct {
	Total  int
	Events []CloudClusterEvent
}

// CloudTopSQLPlan describes one execution plan entry in a TopSQL summary.
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

// CloudTopSQL describes one SQL entry returned by the cloud NGM TopSQL API.
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

// CloudTopSlowQuery describes one aggregated slow-query summary row from cloud
// NGM.
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

// CloudSlowQueryListEntry describes one slow-query sample row from cloud NGM.
type CloudSlowQueryListEntry struct {
	Digest       string
	Query        string
	Timestamp    string
	QueryTime    float64
	MemoryMax    float64
	RequestCount int64
	ConnectionID string
	Raw          map[string]any
}

func catalogEndpoint(orgID, clusterID string) string {
	return fmt.Sprintf(
		"%s/orgs/%s/clusters/%s/data",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
	)
}

func cloudClusterLookupEndpoint() string {
	return cloudClusterLookupPath
}

func clusterDetailEndpoint(orgID, clusterID string) string {
	return fmt.Sprintf(
		"%s/orgs/%s/clusters/%s",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
	)
}

func cloudEventsEndpoint(orgID, clusterID string) string {
	return fmt.Sprintf(
		"%s/activityhub/applications/%s/targets/%s/activities",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
	)
}

func cloudEventDetailEndpoint(orgID, clusterID, eventID string) string {
	return fmt.Sprintf(
		"%s/activityhub/applications/%s/targets/%s/activities/%s",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
		url.PathEscape(strings.TrimSpace(eventID)),
	)
}

func validateCloudTarget(endpoint string, target CloudTarget) error {
	if strings.TrimSpace(target.OrgID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "org id is required"}
	}
	if strings.TrimSpace(target.ClusterID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "cluster id is required"}
	}
	return nil
}

func validateCloudClusterLookup(endpoint string, req CloudClusterLookupRequest) error {
	if strings.TrimSpace(req.ClusterID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "cluster id is required"}
	}
	return nil
}

func validateCloudNGMTarget(endpoint string, target CloudNGMTarget) error {
	if strings.TrimSpace(target.Provider) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "provider is required"}
	}
	if strings.TrimSpace(target.Region) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "region is required"}
	}
	if strings.TrimSpace(target.TenantID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "tenant id is required"}
	}
	if strings.TrimSpace(target.ProjectID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "project id is required"}
	}
	if strings.TrimSpace(target.ClusterID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "cluster id is required"}
	}
	if strings.TrimSpace(target.DeployType) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "deploy type is required"}
	}
	return nil
}

func routingHeaders(ctx RequestContext) http.Header {
	headers := http.Header{}
	headers.Set("X-OrgType", strings.TrimSpace(ctx.OrgType))
	headers.Set("X-OrgID", strings.TrimSpace(ctx.OrgID))
	headers.Set("X-ClusterID", strings.TrimSpace(ctx.ClusterID))
	return headers
}

func ngmHeaders(target CloudNGMTarget) http.Header {
	headers := http.Header{}
	headers.Set("X-Provider", strings.TrimSpace(target.Provider))
	headers.Set("X-Region", strings.TrimSpace(target.Region))
	headers.Set("X-Org-Id", strings.TrimSpace(target.TenantID))
	headers.Set("X-Project-Id", strings.TrimSpace(target.ProjectID))
	headers.Set("X-Cluster-Id", strings.TrimSpace(target.ClusterID))
	headers.Set("X-Deploy-Type", strings.TrimSpace(target.DeployType))
	return headers
}

func requestTraceFromContext(ctx RequestContext, itemID string) requestTrace {
	return requestTrace{
		orgType:   strings.TrimSpace(ctx.OrgType),
		orgID:     strings.TrimSpace(ctx.OrgID),
		clusterID: strings.TrimSpace(ctx.ClusterID),
		itemID:    strings.TrimSpace(itemID),
	}
}

func routeFromKnownCluster(endpoint string, input knownClusterRouteInput) (requestRoute, error) {
	if strings.TrimSpace(input.orgType) == "" {
		return requestRoute{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "org type is required"}
	}
	if strings.TrimSpace(input.orgID) == "" {
		return requestRoute{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "org id is required"}
	}
	if strings.TrimSpace(input.clusterID) == "" {
		return requestRoute{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "cluster id is required"}
	}
	return requestRoute{
		headers: cloneHeader(input.headers),
		trace: requestTrace{
			orgType:   strings.TrimSpace(input.orgType),
			orgID:     strings.TrimSpace(input.orgID),
			clusterID: strings.TrimSpace(input.clusterID),
			itemID:    strings.TrimSpace(input.itemID),
		},
	}, nil
}

func routeFromContext(endpoint string, ctx RequestContext, itemID string) (requestRoute, error) {
	return routeFromKnownCluster(endpoint, knownClusterRouteInput{
		orgType:   ctx.OrgType,
		orgID:     ctx.OrgID,
		clusterID: ctx.ClusterID,
		itemID:    itemID,
		headers:   routingHeaders(ctx),
	})
}

func routeFromItemContext(endpoint string, ctx RequestContext, itemID string) (requestRoute, error) {
	if strings.TrimSpace(itemID) == "" {
		return requestRoute{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "item id is required"}
	}
	return routeFromContext(endpoint, ctx, itemID)
}

func routeFromCloudTarget(endpoint string, target CloudTarget) (requestRoute, error) {
	if err := validateCloudTarget(endpoint, target); err != nil {
		return requestRoute{}, err
	}
	return routeFromKnownCluster(endpoint, knownClusterRouteInput{
		orgType:   "cloud",
		orgID:     target.OrgID,
		clusterID: target.ClusterID,
	})
}

func routeFromCloudClusterLookup(endpoint string, req CloudClusterLookupRequest) (requestRoute, error) {
	if err := validateCloudClusterLookup(endpoint, req); err != nil {
		return requestRoute{}, err
	}
	return requestRoute{
		trace: requestTrace{
			orgType:   "cloud",
			clusterID: strings.TrimSpace(req.ClusterID),
		},
	}, nil
}

func routeFromCloudNGMTarget(endpoint string, target CloudNGMTarget) (requestRoute, error) {
	if err := validateCloudNGMTarget(endpoint, target); err != nil {
		return requestRoute{}, err
	}
	return routeFromKnownCluster(endpoint, knownClusterRouteInput{
		orgType:   "cloud",
		orgID:     target.TenantID,
		clusterID: target.ClusterID,
		headers:   ngmHeaders(target),
	})
}

func asInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int:
		return int64(x), true
	case float64:
		return int64(x), true
	case jsonNumber:
		n, err := strconv.ParseInt(string(x), 10, 64)
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func asFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return n, err == nil
	default:
		return 0, false
	}
}

type jsonNumber string

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneHeader(in http.Header) http.Header {
	if len(in) == 0 {
		return nil
	}
	out := make(http.Header, len(in))
	for key, values := range in {
		out[key] = append([]string(nil), values...)
	}
	return out
}
