package clinicapi

import (
	"github.com/AricSu/tidb-clinic-client/internal/clinic"
	"github.com/AricSu/tidb-clinic-client/internal/model"
	"log"
	"net/http"
	"time"
)

type (
	AuthProvider            = model.AuthProvider
	AuthProviderFunc        = model.AuthProviderFunc
	BearerTokenAuthProvider = model.BearerTokenAuthProvider
	Hooks                   = model.Hooks
	RequestInfo             = model.RequestInfo
	RequestResult           = model.RequestResult
	RequestRetry            = model.RequestRetry
	RequestFailure          = model.RequestFailure
	Config                  = model.Config
	ClientOpt               = clinic.ClientOpt
)

func DefaultConfig() Config {
	return model.DefaultConfig()
}
func WithBearerToken(token string) ClientOpt {
	return clinic.WithBearerToken(token)
}
func WithAuthProvider(provider AuthProvider) ClientOpt {
	return clinic.WithAuthProvider(provider)
}
func WithTimeout(d time.Duration) ClientOpt {
	return clinic.WithTimeout(d)
}
func WithRebuildProbeInterval(d time.Duration) ClientOpt {
	return clinic.WithRebuildProbeInterval(d)
}
func WithVerboseRequestLogs(enabled bool) ClientOpt {
	return clinic.WithVerboseRequestLogs(enabled)
}
func WithRetry(max int, backoff, jitter time.Duration) ClientOpt {
	return clinic.WithRetry(max, backoff, jitter)
}
func WithHTTPClient(hc *http.Client) ClientOpt {
	return clinic.WithHTTPClient(hc)
}
func WithLogger(logger *log.Logger) ClientOpt {
	return clinic.WithLogger(logger)
}
func WithHooks(hooks Hooks) ClientOpt {
	return clinic.WithHooks(hooks)
}
func WithTransportConfig(fn func(*http.Transport)) ClientOpt {
	return clinic.WithTransportConfig(fn)
}
func StaticBearerToken(token string) AuthProvider {
	return model.StaticBearerToken(token)
}
