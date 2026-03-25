package clinicapi

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type transport struct {
	baseURL      *url.URL
	authProvider AuthProvider
	userAgent    string
	httpClient   *http.Client
	retryMax     int
	retryBackoff time.Duration
	retryJitter  time.Duration
	logger       *log.Logger
	hooks        Hooks
}

// CatalogClient exposes catalog endpoints.
type CatalogClient struct{ transport *transport }

// MetricsClient exposes metrics endpoints.
type MetricsClient struct{ transport *transport }

// SlowQueryClient exposes slow query endpoints.
type SlowQueryClient struct{ transport *transport }

// LogClient exposes log search endpoints.
type LogClient struct{ transport *transport }

// ConfigClient exposes config snapshot endpoints.
type ConfigClient struct{ transport *transport }

// CloudClient exposes cloud-only known-target endpoints.
type CloudClient struct{ transport *transport }

// OPClient reserves the namespace for future On-Premise-only endpoints.
type OPClient struct{ transport *transport }

// Client is the root Clinic SDK client.
type Client struct {
	cfg Config

	transport   *transport
	Catalog     *CatalogClient
	Metrics     *MetricsClient
	SlowQueries *SlowQueryClient
	Logs        *LogClient
	Configs     *ConfigClient
	Cloud       *CloudClient
	OP          *OPClient
}

// NewClient constructs a Client from a base URL plus option functions.
func NewClient(baseURL string, opts ...ClientOpt) (*Client, error) {
	cfg := DefaultConfig()
	cfg.BaseURL = strings.TrimSpace(baseURL)
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return NewClientWithConfig(cfg)
}

// NewClientWithConfig constructs a Client from a complete Config value.
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
	if strings.TrimSpace(cfg.UserAgent) != "" {
		merged.UserAgent = strings.TrimSpace(cfg.UserAgent)
	}
	if cfg.Timeout > 0 {
		merged.Timeout = cfg.Timeout
	}
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
		baseURL:      parsedURL,
		authProvider: merged.AuthProvider,
		userAgent:    merged.UserAgent,
		httpClient:   hc,
		retryMax:     merged.RetryMax,
		retryBackoff: merged.RetryBackoff,
		retryJitter:  merged.RetryJitter,
		logger:       merged.Logger,
		hooks:        merged.Hooks,
	}
	client := &Client{
		cfg:         merged,
		transport:   t,
		Catalog:     &CatalogClient{transport: t},
		Metrics:     &MetricsClient{transport: t},
		SlowQueries: &SlowQueryClient{transport: t},
		Logs:        &LogClient{transport: t},
		Configs:     &ConfigClient{transport: t},
		Cloud:       &CloudClient{transport: t},
		OP:          &OPClient{transport: t},
	}
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

// Config returns the normalized client configuration snapshot.
func (c *Client) Config() Config {
	if c == nil {
		return Config{}
	}
	return c.cfg
}

// Close releases any client-owned resources.
func (c *Client) Close() error {
	return nil
}
