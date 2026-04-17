package clinicapi

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type transport struct {
	baseURL              *url.URL
	authProvider         AuthProvider
	httpClient           *http.Client
	rebuildProbeInterval time.Duration
	verboseRequestLogs   bool
	retryMax             int
	retryBackoff         time.Duration
	retryJitter          time.Duration
	logger               *log.Logger
	hooks                Hooks
}
type catalogClient struct {
	transport *transport
	client    *Client
}
type metricsAPIClient struct {
	transport *transport
	client    *Client
}
type dataProxyClient struct {
	transport *transport
	client    *Client
}
type slowQueryClient struct {
	transport *transport
	client    *Client
}
type logSearchAPIClient struct {
	transport *transport
	client    *Client
}
type lokiClient struct {
	transport *transport
	client    *Client
}
type configClient struct {
	transport *transport
	client    *Client
}
type cloudClient struct {
	transport *transport
	client    *Client
}
type Client struct {
	cfg          Config
	transport    *transport
	catalog      *catalogClient
	metricsAPI   *metricsAPIClient
	dataProxy    *dataProxyClient
	slowQueries  *slowQueryClient
	logSearchAPI *logSearchAPIClient
	loki         *lokiClient
	configsAPI   *configClient
	cloud        *cloudClient
}

func NewClientWithConfig(cfg Config) (*Client, error) {
	merged := DefaultConfig()
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
	merged.AuthProvider = buildAuthProvider(merged)
	merged.BearerToken = ""
	parsedURL, err := url.Parse(merged.BaseURL)
	if err != nil {
		return nil, err
	}
	hc := merged.HTTPClient
	if hc == nil {
		hc = &http.Client{
			Timeout: merged.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        merged.MaxIdleConns,
				MaxIdleConnsPerHost: merged.MaxIdlePerHost,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: merged.TLSHandshake,
				DisableKeepAlives:   merged.DisableKeepAlive,
			},
		}
	} else if hc.Timeout <= 0 {
		hc.Timeout = merged.Timeout
	}
	t := &transport{
		baseURL:              parsedURL,
		authProvider:         merged.AuthProvider,
		httpClient:           hc,
		rebuildProbeInterval: merged.RebuildProbeInterval,
		verboseRequestLogs:   merged.VerboseRequestLogs,
		retryMax:             merged.RetryMax,
		retryBackoff:         merged.RetryBackoff,
		retryJitter:          merged.RetryJitter,
		logger:               merged.Logger,
		hooks:                merged.Hooks,
	}
	client := &Client{cfg: merged, transport: t}
	client.catalog = &catalogClient{transport: t, client: client}
	client.metricsAPI = &metricsAPIClient{transport: t, client: client}
	client.dataProxy = &dataProxyClient{transport: t, client: client}
	client.slowQueries = &slowQueryClient{transport: t, client: client}
	client.logSearchAPI = &logSearchAPIClient{transport: t, client: client}
	client.loki = &lokiClient{transport: t, client: client}
	client.configsAPI = &configClient{transport: t, client: client}
	client.cloud = &cloudClient{transport: t, client: client}
	return client, nil
}
func buildAuthProvider(cfg Config) AuthProvider {
	if cfg.AuthProvider != nil {
		return cfg.AuthProvider
	}
	if strings.TrimSpace(cfg.BearerToken) == "" {
		return nil
	}
	return StaticBearerToken(cfg.BearerToken)
}
func (c *Client) Close() error {
	return nil
}
func (c *Client) ListCatalogData(ctx context.Context, req ListClusterDataRequest) ([]ClinicDataItem, error) {
	return c.catalog.ListClusterData(ctx, req)
}
func (c *Client) EnsureCatalogDataReadable(ctx context.Context, requestContext RequestContext, item ClinicDataItem) error {
	return c.catalog.EnsureCatalogDataReadable(ctx, requestContext, item)
}
func (c *Client) QueryRange(ctx context.Context, req MetricsQueryRangeRequest) (MetricQueryRangeResult, error) {
	return c.metricsAPI.QueryRange(ctx, req)
}
func (c *Client) QueryRangeWithAutoSplit(ctx context.Context, req MetricsQueryRangeRequest) (MetricQueryRangeResult, error) {
	return c.metricsAPI.QueryRangeWithAutoSplit(ctx, req)
}
func (c *Client) QueryInstant(ctx context.Context, req MetricsQueryInstantRequest) (MetricQueryInstantResult, error) {
	return c.metricsAPI.QueryInstant(ctx, req)
}
func (c *Client) QuerySeries(ctx context.Context, req MetricsQuerySeriesRequest) (MetricQuerySeriesResult, error) {
	return c.metricsAPI.QuerySeries(ctx, req)
}
func (c *Client) QuerySQL(ctx context.Context, req DataProxyQueryRequest) (DataProxyQueryResult, error) {
	return c.dataProxy.Query(ctx, req)
}
func (c *Client) SchemaSQL(ctx context.Context, req DataProxySchemaRequest) (DataProxySchemaResult, error) {
	return c.dataProxy.Schema(ctx, req)
}
func (c *Client) QueryLogs(ctx context.Context, req LokiQueryRequest) (LokiQueryResult, error) {
	return c.loki.Query(ctx, req)
}
func (c *Client) QueryLogsRange(ctx context.Context, req LokiQueryRangeRequest) (LokiQueryResult, error) {
	return c.loki.QueryRange(ctx, req)
}
func (c *Client) LogLabels(ctx context.Context, req LokiLabelsRequest) (LokiLabelsResult, error) {
	return c.loki.Labels(ctx, req)
}
func (c *Client) LogLabelValues(ctx context.Context, req LokiLabelValuesRequest) (LokiLabelsResult, error) {
	return c.loki.LabelValues(ctx, req)
}
func (c *Client) SearchLogs(ctx context.Context, req LogSearchRequest) (LogSearchResult, error) {
	return c.logSearchAPI.Search(ctx, req)
}
func (c *Client) SlowQueryRecords(ctx context.Context, req SlowQueryRequest) (SlowQueryRecordsResult, error) {
	return c.slowQueries.Query(ctx, req)
}
func (c *Client) ListCollectedSlowQueries(ctx context.Context, req CollectedSlowQueryListRequest) ([]CloudSlowQueryListEntry, error) {
	return c.slowQueries.List(ctx, req)
}
func (c *Client) GetCollectedSlowQueryDetail(ctx context.Context, req CollectedSlowQueryDetailRequest) (map[string]any, error) {
	return c.slowQueries.GetDetail(ctx, req)
}
func (c *Client) GetConfig(ctx context.Context, req ConfigRequest) (ConfigResult, error) {
	return c.configsAPI.Get(ctx, req)
}
func (c *Client) SearchClusters(ctx context.Context, req CloudClusterSearchRequest) ([]CloudCluster, error) {
	return c.cloud.SearchClusters(ctx, req)
}
func (c *Client) GetCluster(ctx context.Context, req CloudClusterLookupRequest) (CloudCluster, error) {
	return c.cloud.GetCluster(ctx, req)
}
func (c *Client) GetOrg(ctx context.Context, req OrgRequest) (Org, error) {
	return c.cloud.GetOrg(ctx, req)
}
func (c *Client) GetClusterDetail(ctx context.Context, req CloudClusterDetailRequest) (CloudClusterDetail, error) {
	return c.cloud.GetClusterDetail(ctx, req)
}
func (c *Client) GetTopology(ctx context.Context, req CloudClusterTopologyRequest) (CloudClusterDetail, error) {
	return c.cloud.GetTopology(ctx, req)
}
func (c *Client) QueryEvents(ctx context.Context, req CloudEventsRequest) (CloudEventsResult, error) {
	return c.cloud.QueryEvents(ctx, req)
}
func (c *Client) GetEventDetail(ctx context.Context, req CloudEventDetailRequest) (map[string]any, error) {
	return c.cloud.GetEventDetail(ctx, req)
}
func (c *Client) GetTopSQLSummary(ctx context.Context, req CloudTopSQLSummaryRequest) ([]CloudTopSQL, error) {
	return c.cloud.GetTopSQLSummary(ctx, req)
}
func (c *Client) GetTopSlowQueries(ctx context.Context, req CloudTopSlowQueriesRequest) ([]CloudTopSlowQuery, error) {
	return c.cloud.GetTopSlowQueries(ctx, req)
}
func (c *Client) ListSlowQueries(ctx context.Context, req CloudSlowQueryListRequest) ([]CloudSlowQueryListEntry, error) {
	return c.cloud.ListSlowQueries(ctx, req)
}
func (c *Client) GetSlowQueryDetail(ctx context.Context, req CloudSlowQueryDetailRequest) (map[string]any, error) {
	return c.cloud.GetSlowQueryDetail(ctx, req)
}
func (c *Client) ListProfileGroups(ctx context.Context, req CloudProfileGroupsRequest) (CloudProfileGroupsResult, error) {
	return c.cloud.ListProfileGroups(ctx, req)
}
func (c *Client) GetProfileGroupDetail(ctx context.Context, req CloudProfileGroupDetailRequest) (CloudProfileGroupDetail, error) {
	return c.cloud.GetProfileGroupDetail(ctx, req)
}
func (c *Client) GetProfileActionToken(ctx context.Context, req CloudProfileActionTokenRequest) (string, error) {
	return c.cloud.GetProfileActionToken(ctx, req)
}
func (c *Client) DownloadProfile(ctx context.Context, req CloudProfileDownloadRequest) (CloudDownloadedArtifact, error) {
	return c.cloud.DownloadProfile(ctx, req)
}
func (c *Client) FetchProfile(ctx context.Context, req CloudProfileFetchRequest) (CloudDownloadedArtifact, error) {
	return c.cloud.FetchProfile(ctx, req)
}
func (c *Client) ListPlanReplayerArtifacts(ctx context.Context, req CloudDiagnosticListRequest) (CloudDiagnosticListResult, error) {
	return c.cloud.ListPlanReplayerArtifacts(ctx, req)
}
func (c *Client) ListOOMRecordArtifacts(ctx context.Context, req CloudDiagnosticListRequest) (CloudDiagnosticListResult, error) {
	return c.cloud.ListOOMRecordArtifacts(ctx, req)
}
func (c *Client) DownloadDiagnosticArtifact(ctx context.Context, req CloudDiagnosticDownloadRequest) (CloudDownloadedArtifact, error) {
	return c.cloud.DownloadDiagnosticArtifact(ctx, req)
}
