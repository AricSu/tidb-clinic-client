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
type SlowQueryClient struct{ handle *ClusterHandle }
type CollectedDataClient struct{ handle *ClusterHandle }
type ProfilingClient struct{ handle *ClusterHandle }
type DiagnosticsClient struct{ handle *ClusterHandle }
type Client struct {
	cfg      model.Config
	clinic   clinicService
	Clusters *ClustersClient
}

func NewClient(baseURL string, opts ...ClientOpt) (*Client, error) {
	cfg := model.MergeConfig(model.Config{})
	cfg.BaseURL = strings.TrimSpace(baseURL)
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return NewClientWithConfig(cfg)
}
func NewClientWithConfig(cfg model.Config) (*Client, error) {
	merged := model.MergeConfig(cfg)
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
	ClusterDetail(ctx context.Context, target controlPlaneTarget) (apitypes.CloudClusterDetail, error)
}
type catalogService interface {
	ListCatalogData(ctx context.Context, requestContext apitypes.RequestContext) ([]apitypes.ClinicDataItem, error)
	EnsureCatalogDataReadable(ctx context.Context, requestContext apitypes.RequestContext, item apitypes.ClinicDataItem, dataType apitypes.CatalogDataType) error
	DownloadCollectedData(ctx context.Context, requestContext apitypes.RequestContext, item apitypes.ClinicDataItem) (DownloadedArtifact, error)
}
type metricsService interface {
	QueryRange(ctx context.Context, target metricsTarget, query TimeSeriesQuery) (MetricQueryRangeResult, error)
	CompileRange(ctx context.Context, target metricsTarget, query MetricsCompileQuery) ([]CompiledTimeseriesDigest, error)
}
type logsService interface {
	QueryLogsRange(ctx context.Context, target logsTarget, query LogRangeQuery) (apitypes.LokiQueryResult, error)
	LogLabels(ctx context.Context, target logsTarget, query LogLabelsQuery) (apitypes.LokiLabelsResult, error)
	LogLabelValues(ctx context.Context, target logsTarget, query LogLabelValuesQuery) (apitypes.LokiLabelsResult, error)
}
type slowQueryService interface {
	QueryCloudSlowQueries(ctx context.Context, target apitypes.CloudNGMTarget, query SlowQueryQuery) (SlowQueryResult, error)
	QueryCloudSlowQuerySamples(ctx context.Context, target apitypes.CloudNGMTarget, query SlowQuerySamplesQuery, startTime, endTime int64) (SlowQuerySamplesResult, error)
	QuerySlowQueries(ctx context.Context, requestContext apitypes.RequestContext, item apitypes.ClinicDataItem, query SlowQueryQuery) (SlowQueryResult, error)
	QuerySlowQuerySamples(ctx context.Context, requestContext apitypes.RequestContext, item apitypes.ClinicDataItem, query SlowQuerySamplesQuery, startTime, endTime int64) (SlowQuerySamplesResult, error)
}
type profilingService interface {
	ListProfileGroups(ctx context.Context, target profilingTarget, beginTime, endTime int64) (apitypes.CloudProfileGroupsResult, error)
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
	slowQueryService
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
