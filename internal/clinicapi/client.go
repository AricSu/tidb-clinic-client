package clinicapi

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AricSu/tidb-clinic-client/internal/model"
)

type transport struct {
	baseURL              *url.URL
	authProvider         AuthProvider
	httpClient           *http.Client
	rebuildProbeInterval time.Duration
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
type lokiClient struct {
	transport *transport
	client    *Client
}
type cloudClient struct {
	transport *transport
	client    *Client
}
type Client struct {
	cfg        Config
	transport  *transport
	catalog    *catalogClient
	metricsAPI *metricsAPIClient
	loki       *lokiClient
	cloud      *cloudClient
}

func NewClientWithConfig(cfg Config) (*Client, error) {
	merged := model.MergeConfig(cfg)
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
			Transport: &http.Transport{
				MaxIdleConns:        merged.MaxIdleConns,
				MaxIdleConnsPerHost: merged.MaxIdlePerHost,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: merged.TLSHandshake,
				DisableKeepAlives:   merged.DisableKeepAlive,
			},
		}
	}
	t := &transport{
		baseURL:              parsedURL,
		authProvider:         merged.AuthProvider,
		httpClient:           hc,
		rebuildProbeInterval: merged.RebuildProbeInterval,
		retryMax:             merged.RetryMax,
		retryBackoff:         merged.RetryBackoff,
		retryJitter:          merged.RetryJitter,
		logger:               merged.Logger,
		hooks:                merged.Hooks,
	}
	client := &Client{cfg: merged, transport: t}
	client.catalog = &catalogClient{transport: t, client: client}
	client.metricsAPI = &metricsAPIClient{transport: t, client: client}
	client.loki = &lokiClient{transport: t, client: client}
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
func (c *Client) EnsureCatalogDataReadable(ctx context.Context, req EnsureCatalogDataReadableRequest) error {
	return c.catalog.EnsureCatalogDataReadable(ctx, req)
}
func (c *Client) DownloadCollectedData(ctx context.Context, req CollectedDataDownloadRequest) ([]byte, error) {
	return c.catalog.DownloadCollectedData(ctx, req)
}
func (c *Client) QueryRange(ctx context.Context, req MetricsQueryRangeRequest) (MetricQueryRangeResult, error) {
	return c.metricsAPI.QueryRange(ctx, req)
}
func (c *Client) QueryRangeWithAutoSplit(ctx context.Context, req MetricsQueryRangeRequest) (MetricQueryRangeResult, error) {
	return c.metricsAPI.QueryRangeWithAutoSplit(ctx, req)
}
func (c *Client) QuerySlowQueries(ctx context.Context, req SlowQueryRequest) (SlowQueryResult, error) {
	return c.catalog.QuerySlowQueries(ctx, req)
}
func (c *Client) QuerySlowQuerySamples(ctx context.Context, req SlowQuerySamplesRequest) (SlowQuerySamplesResult, error) {
	return c.catalog.QuerySlowQuerySamples(ctx, req)
}
func (c *Client) QueryCloudSlowQueries(ctx context.Context, req CloudSlowQueryRequest) (SlowQueryResult, error) {
	return c.cloud.QuerySlowQueries(ctx, req)
}
func (c *Client) QueryCloudSlowQuerySamples(ctx context.Context, req CloudSlowQueryRequest) (SlowQuerySamplesResult, error) {
	return c.cloud.QuerySlowQuerySamples(ctx, req)
}
func (c *Client) QueryCloudSlowQueryDetail(ctx context.Context, req CloudSlowQueryDetailRequest) (map[string]any, error) {
	return c.cloud.QuerySlowQueryDetail(ctx, req)
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
func (c *Client) GetCluster(ctx context.Context, req CloudClusterLookupRequest) (CloudCluster, error) {
	return c.cloud.GetCluster(ctx, req)
}
func (c *Client) GetOrg(ctx context.Context, req OrgRequest) (Org, error) {
	return c.cloud.GetOrg(ctx, req)
}
func (c *Client) GetClusterDetail(ctx context.Context, req CloudClusterDetailRequest) (CloudClusterDetail, error) {
	return c.cloud.GetClusterDetail(ctx, req)
}
func (c *Client) GetClusterDetailRaw(ctx context.Context, req CloudClusterDetailRequest) (map[string]any, error) {
	return c.cloud.GetClusterDetailRaw(ctx, req)
}
func (c *Client) QueryEventsRaw(ctx context.Context, req CloudEventsRequest) (map[string]any, error) {
	return c.cloud.QueryEventsRaw(ctx, req)
}
func (c *Client) ListProfileGroups(ctx context.Context, req CloudProfileGroupsRequest) (CloudProfileGroupsResult, error) {
	return c.cloud.ListProfileGroups(ctx, req)
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
