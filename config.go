package clinicapi

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultUserAgent = "tidb-clinic-client"

// Config configures a Clinic API client.
type Config struct {
	BaseURL          string
	BearerToken      string
	AuthProvider     AuthProvider
	UserAgent        string
	Timeout          time.Duration
	RetryMax         int
	RetryBackoff     time.Duration
	RetryJitter      time.Duration
	MaxIdleConns     int
	MaxIdlePerHost   int
	TLSHandshake     time.Duration
	DisableKeepAlive bool
	HTTPClient       *http.Client
	Logger           *log.Logger
	Hooks            Hooks
}

// ClientOpt mutates a Config before client construction.
type ClientOpt func(*Config)

// DefaultConfig returns a production-oriented default configuration.
func DefaultConfig() Config {
	return Config{
		UserAgent:      defaultUserAgent,
		Timeout:        20 * time.Second,
		RetryMax:       2,
		RetryBackoff:   250 * time.Millisecond,
		RetryJitter:    250 * time.Millisecond,
		MaxIdleConns:   64,
		MaxIdlePerHost: 16,
		TLSHandshake:   10 * time.Second,
	}
}

// Valid validates the configuration for client construction.
func (c Config) Valid() error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return errors.New("clinic api base URL is required")
	}
	u, err := url.Parse(strings.TrimSpace(c.BaseURL))
	if err != nil {
		return err
	}
	if scheme := strings.ToLower(strings.TrimSpace(u.Scheme)); scheme != "http" && scheme != "https" {
		return errors.New("clinic api base URL must use http or https")
	}
	if strings.TrimSpace(c.BearerToken) == "" && c.AuthProvider == nil {
		return errors.New("clinic api bearer token or auth provider is required")
	}
	if c.Timeout <= 0 {
		return errors.New("clinic api timeout must be positive")
	}
	if c.RetryMax < 0 {
		return errors.New("clinic api retry max must be non-negative")
	}
	if c.RetryBackoff < 0 || c.RetryJitter < 0 {
		return errors.New("clinic api retry backoff and jitter must be non-negative")
	}
	return nil
}

// WithBearerToken sets the static bearer token convenience field.
func WithBearerToken(token string) ClientOpt {
	return func(cfg *Config) {
		cfg.BearerToken = ""
		cfg.AuthProvider = nil
		if trimmed := strings.TrimSpace(token); trimmed != "" {
			cfg.AuthProvider = StaticBearerToken(trimmed)
		}
	}
}

// WithAuthProvider sets a custom auth provider.
func WithAuthProvider(provider AuthProvider) ClientOpt {
	return func(cfg *Config) {
		cfg.AuthProvider = provider
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) ClientOpt {
	return func(cfg *Config) {
		cfg.Timeout = d
	}
}

// WithRetry sets retry policy values.
func WithRetry(max int, backoff, jitter time.Duration) ClientOpt {
	return func(cfg *Config) {
		cfg.RetryMax = max
		cfg.RetryBackoff = backoff
		cfg.RetryJitter = jitter
	}
}

// WithUserAgent sets the User-Agent header value.
func WithUserAgent(ua string) ClientOpt {
	return func(cfg *Config) {
		cfg.UserAgent = ua
	}
}

// WithHTTPClient injects a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOpt {
	return func(cfg *Config) {
		cfg.HTTPClient = hc
	}
}

// WithLogger injects a standard-library logger for request lifecycle logging.
func WithLogger(logger *log.Logger) ClientOpt {
	return func(cfg *Config) {
		cfg.Logger = logger
	}
}

// WithHooks injects request lifecycle hooks.
func WithHooks(hooks Hooks) ClientOpt {
	return func(cfg *Config) {
		cfg.Hooks = hooks
	}
}

// WithTransportConfig mutates the underlying HTTP transport configuration.
func WithTransportConfig(fn func(*http.Transport)) ClientOpt {
	return func(cfg *Config) {
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
