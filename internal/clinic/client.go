package clinic

import (
	"context"
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
	"github.com/AricSu/tidb-clinic-client/internal/model"
	"log"
	"net/http"
	"strings"
	"time"
)

type ClustersClient struct{ client *Client }
type MetricsClient struct{ handle *ClusterHandle }
type LogClient struct{ handle *ClusterHandle }
type SQLAnalyticsClient struct{ handle *ClusterHandle }
type ConfigsClient struct{ handle *ClusterHandle }
type ProfilingClient struct{ handle *ClusterHandle }
type DiagnosticsClient struct{ handle *ClusterHandle }
type CapabilitiesClient struct{ handle *ClusterHandle }
type Client struct {
	cfg      model.Config
	clinic   clinicService
	Clusters *ClustersClient
}

func NewClient(baseURL string, opts ...ClientOpt) (*Client, error) {
	cfg := model.DefaultConfig()
	cfg.BaseURL = strings.TrimSpace(baseURL)
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return NewClientWithConfig(cfg)
}
func NewClientWithConfig(cfg model.Config) (*Client, error) {
	merged := model.DefaultConfig()
	if strings.TrimSpace(cfg.BaseURL) != "" {
		merged.BaseURL = strings.TrimSpace(cfg.BaseURL)
	}
	if strings.TrimSpace(cfg.BearerToken) != "" {
		merged.BearerToken = strings.TrimSpace(cfg.BearerToken)
	}
	if cfg.AuthProvider != nil {
		merged.AuthProvider = cfg.AuthProvider
	}
	if cfg.Timeout > 0 {
		merged.Timeout = cfg.Timeout
	}
	if cfg.RebuildProbeInterval > 0 {
		merged.RebuildProbeInterval = cfg.RebuildProbeInterval
	}
	merged.VerboseRequestLogs = cfg.VerboseRequestLogs
	if cfg.RetryMax != 0 {
		merged.RetryMax = cfg.RetryMax
	}
	if cfg.RetryBackoff != 0 {
		merged.RetryBackoff = cfg.RetryBackoff
	}
	if cfg.RetryJitter != 0 {
		merged.RetryJitter = cfg.RetryJitter
	}
	if cfg.MaxIdleConns != 0 {
		merged.MaxIdleConns = cfg.MaxIdleConns
	}
	if cfg.MaxIdlePerHost != 0 {
		merged.MaxIdlePerHost = cfg.MaxIdlePerHost
	}
	if cfg.TLSHandshake != 0 {
		merged.TLSHandshake = cfg.TLSHandshake
	}
	merged.DisableKeepAlive = cfg.DisableKeepAlive
	if cfg.HTTPClient != nil {
		merged.HTTPClient = cfg.HTTPClient
	}
	if cfg.Logger != nil {
		merged.Logger = cfg.Logger
	}
	merged.Hooks = cfg.Hooks
	if err := merged.Valid(); err != nil {
		return nil, err
	}
	client := &Client{cfg: merged}
	client.Clusters = &ClustersClient{client: client}
	clinic, err := newClinicServiceClient(client)
	if err != nil {
		return nil, err
	}
	client.clinic = clinic
	return client, nil
}
func (c *Client) Config() model.Config {
	if c == nil {
		return model.Config{}
	}
	return c.cfg
}
func (c *Client) Close() error {
	return nil
}

type ClientOpt func(*model.Config)

func WithBearerToken(token string) ClientOpt {
	return func(cfg *model.Config) {
		cfg.BearerToken = ""
		cfg.AuthProvider = nil
		if trimmed := strings.TrimSpace(token); trimmed != "" {
			cfg.AuthProvider = model.StaticBearerToken(trimmed)
		}
	}
}
func WithAuthProvider(provider model.AuthProvider) ClientOpt {
	return func(cfg *model.Config) {
		cfg.AuthProvider = provider
	}
}
func WithTimeout(d time.Duration) ClientOpt {
	return func(cfg *model.Config) {
		cfg.Timeout = d
	}
}
func WithRebuildProbeInterval(d time.Duration) ClientOpt {
	return func(cfg *model.Config) {
		cfg.RebuildProbeInterval = d
	}
}
func WithVerboseRequestLogs(enabled bool) ClientOpt {
	return func(cfg *model.Config) {
		cfg.VerboseRequestLogs = enabled
	}
}
func WithRetry(max int, backoff, jitter time.Duration) ClientOpt {
	return func(cfg *model.Config) {
		cfg.RetryMax = max
		cfg.RetryBackoff = backoff
		cfg.RetryJitter = jitter
	}
}
func WithHTTPClient(hc *http.Client) ClientOpt {
	return func(cfg *model.Config) {
		cfg.HTTPClient = hc
	}
}
func WithLogger(logger *log.Logger) ClientOpt {
	return func(cfg *model.Config) {
		cfg.Logger = logger
	}
}
func WithHooks(hooks model.Hooks) ClientOpt {
	return func(cfg *model.Config) {
		cfg.Hooks = hooks
	}
}
func WithTransportConfig(fn func(*http.Transport)) ClientOpt {
	return func(cfg *model.Config) {
		if cfg.HTTPClient == nil {
			cfg.HTTPClient = &http.Client{}
		}
		base, _ := cfg.HTTPClient.Transport.(*http.Transport)
		if base == nil {
			base = &http.Transport{}
		} else {
			base = base.Clone()
		}
		fn(base)
		cfg.HTTPClient.Transport = base
	}
}

type clusterResolver interface {
	ResolveCluster(ctx context.Context, selector ClusterSelector) (apitypes.CloudCluster, error)
	ResolveOrg(ctx context.Context, orgID string) (apitypes.Org, error)
	SearchClusters(ctx context.Context, query ClusterSearchQuery) ([]apitypes.CloudCluster, error)
	ClusterDetail(ctx context.Context, target controlPlaneTarget) (apitypes.CloudClusterDetail, error)
	Topology(ctx context.Context, target resolvedClusterTarget) (apitypes.CloudClusterDetail, error)
	Events(ctx context.Context, target controlPlaneTarget, startTime, endTime int64) (apitypes.CloudEventsResult, error)
	EventDetail(ctx context.Context, target controlPlaneTarget, eventID string) (map[string]any, error)
}
type catalogService interface {
	ListCatalogData(ctx context.Context, requestContext apitypes.RequestContext) ([]apitypes.ClinicDataItem, error)
	EnsureCatalogDataReadable(ctx context.Context, requestContext apitypes.RequestContext, item apitypes.ClinicDataItem) error
}
type metricsService interface {
	QueryRange(ctx context.Context, target metricsTarget, query TimeSeriesQuery) (MetricQueryRangeResult, error)
	QueryInstant(ctx context.Context, target metricsTarget, query TimeSeriesQuery) (MetricQueryInstantResult, error)
	QuerySeries(ctx context.Context, target metricsTarget, query TimeSeriesQuery) (MetricQuerySeriesResult, error)
}
type logsService interface {
	QueryLogs(ctx context.Context, target logsTarget, query LogQuery) (apitypes.LokiQueryResult, error)
	QueryLogsRange(ctx context.Context, target logsTarget, query LogRangeQuery) (apitypes.LokiQueryResult, error)
	LogLabels(ctx context.Context, target logsTarget, query LogLabelsQuery) (apitypes.LokiLabelsResult, error)
	LogLabelValues(ctx context.Context, target logsTarget, query LogLabelValuesQuery) (apitypes.LokiLabelsResult, error)
	SearchLogs(ctx context.Context, requestContext apitypes.RequestContext, itemID string, query LogSearchQuery) (LogSearchResult, error)
}
type sqlAnalyticsService interface {
	QuerySQL(ctx context.Context, target sqlTarget, query SQLQuery) (apitypes.DataProxyQueryResult, error)
	SchemaSQL(ctx context.Context, target sqlTarget, query SchemaQuery) (apitypes.DataProxySchemaResult, error)
	TopSQLSummary(ctx context.Context, target resolvedClusterTarget, query TopSQLSummaryQuery) ([]apitypes.CloudTopSQL, error)
	TopSlowQueries(ctx context.Context, target resolvedClusterTarget, query TopSlowQueriesQuery) ([]apitypes.CloudTopSlowQuery, error)
	SlowQuerySamples(ctx context.Context, target resolvedClusterTarget, query SlowQuerySamplesQuery) ([]apitypes.CloudSlowQueryListEntry, error)
	SlowQueryDetail(ctx context.Context, target resolvedClusterTarget, query SlowQueryDetailQuery) (map[string]any, error)
	SQLStatements(ctx context.Context, target sqlTarget, query SQLStatementsQuery) (apitypes.DataProxyQueryResult, error)
	SlowQueryRecords(ctx context.Context, requestContext apitypes.RequestContext, itemID string, query SlowQueryRecordsQuery) (apitypes.SlowQueryRecordsResult, error)
}
type configsService interface {
	GetConfig(ctx context.Context, requestContext apitypes.RequestContext, itemID string, query ConfigQuery) (ConfigResult, error)
}
type profilingService interface {
	ListProfileGroups(ctx context.Context, target profilingTarget, beginTime, endTime int64) (apitypes.CloudProfileGroupsResult, error)
	ProfileDetail(ctx context.Context, target profilingTarget, timestamp int64) (apitypes.CloudProfileGroupDetail, error)
	ProfileActionToken(ctx context.Context, target profilingTarget, req ProfileActionTokenRequest) (string, error)
	DownloadProfile(ctx context.Context, target profilingTarget, req ProfileDownloadRequest) (DownloadedArtifact, error)
	FetchProfile(ctx context.Context, target profilingTarget, req ProfileFetchRequest) (DownloadedArtifact, error)
}
type diagnosticsService interface {
	ListPlanReplayer(ctx context.Context, target diagnosticsTarget, start, end int64) (apitypes.CloudDiagnosticListResult, error)
	ListOOMRecord(ctx context.Context, target diagnosticsTarget, start, end int64) (apitypes.CloudDiagnosticListResult, error)
	DownloadDiagnostic(ctx context.Context, target diagnosticsTarget, req DiagnosticDownloadRequest) (DownloadedArtifact, error)
}
type clinicService interface {
	clusterResolver
	catalogService
	metricsService
	logsService
	sqlAnalyticsService
	configsService
	profilingService
	diagnosticsService
}
type clinicServiceClient struct {
	api    *apitypes.Client
	client *Client
}

func newClinicServiceClient(client *Client) (clinicService, error) {
	api, err := apitypes.NewClientWithConfig(clinicAPIConfig(client.cfg))
	if err != nil {
		return nil, err
	}
	return &clinicServiceClient{api: api, client: client}, nil
}
