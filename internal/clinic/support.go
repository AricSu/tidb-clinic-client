package clinic

import (
	"errors"
	"fmt"
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
	"github.com/AricSu/tidb-clinic-client/internal/model"
	"sort"
	"strings"
)

type (
	AuthProvider                   = model.AuthProvider
	AuthProviderFunc               = model.AuthProviderFunc
	BearerTokenAuthProvider        = model.BearerTokenAuthProvider
	ErrorClass                     = model.ErrorClass
	Error                          = model.Error
	Hooks                          = model.Hooks
	RequestInfo                    = model.RequestInfo
	RequestResult                  = model.RequestResult
	RequestRetry                   = model.RequestRetry
	RequestFailure                 = model.RequestFailure
	RetainedDataRef                = model.RetainedDataRef
	CollectedDataItem              = model.CollectedDataItem
	QueryMetadata                  = model.QueryMetadata
	TimeSeriesQuery                = model.TimeSeriesQuery
	MetricsCompileQuery            = model.MetricsCompileQuery
	LogRangeQuery                  = model.LogRangeQuery
	SlowQueryQuery                 = model.SlowQueryQuery
	SlowQuerySamplesQuery          = model.SlowQuerySamplesQuery
	LogLabelsQuery                 = model.LogLabelsQuery
	LogLabelValuesQuery            = model.LogLabelValuesQuery
	ProfileFetchRequest            = model.ProfileFetchRequest
	DiagnosticDownloadRequest      = model.DiagnosticDownloadRequest
	CollectedDataDownloadRequest   = model.CollectedDataDownloadRequest
	SeriesKind                     = model.SeriesKind
	SeriesPoint                    = model.SeriesPoint
	Series                         = model.Series
	SeriesResult                   = model.SeriesResult
	StreamValue                    = model.StreamValue
	SlowQueryRecord                = model.SlowQueryRecord
	SlowQueryResult                = model.SlowQueryResult
	Stream                         = model.Stream
	StreamResult                   = model.StreamResult
	ListResult                     = model.ListResult
	BlobResult                     = model.BlobResult
	MetricQueryRangeResult         = model.SeriesResult
	CompiledTimeseriesProblemRange = model.CompiledTimeseriesProblemRange
	CompiledTimeseriesEvent        = model.CompiledTimeseriesEvent
	CompiledTimeseriesDigest       = model.CompiledTimeseriesDigest
	LogQueryResult                 = model.StreamResult
	LogLabelsResult                = model.ListResult
	SlowQuerySamplesResult         = model.ListResult
	ProfileGroupsResult            = model.ListResult
	DiagnosticListResult           = model.ListResult
	DownloadedArtifact             = model.DownloadedArtifact
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
func compactNonEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
func stringifyAny(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
func canonicalTargetKey(target resolvedTarget) string {
	return strings.Join([]string{
		strings.TrimSpace(string(target.Platform)),
		strings.TrimSpace(target.OrgID),
		strings.TrimSpace(target.ClusterID),
	}, "|")
}
func resolutionAmbiguityMessage(clusterID string, candidates []resolvedTarget) string {
	labels := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		labels = append(labels, fmt.Sprintf("%s(org=%s)", candidate.Platform, candidate.OrgID))
	}
	sort.Strings(labels)
	return fmt.Sprintf(
		"cluster id %s resolved to multiple targets: %s; specify platform or org id",
		strings.TrimSpace(clusterID),
		strings.Join(labels, ", "),
	)
}
func cloneResolvedClusterTarget(in *resolvedClusterTarget) *resolvedClusterTarget {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
func cloneResolvedTiUPTarget(in *resolvedTiUPTarget) *resolvedTiUPTarget {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
func cloneResolvedTarget(in resolvedTarget) resolvedTarget {
	out := in
	out.Cloud = cloneResolvedClusterTarget(in.Cloud)
	out.TiUP = cloneResolvedTiUPTarget(in.TiUP)
	return out
}
func clinicAPIConfig(cfg model.Config) apitypes.Config {
	return apitypes.Config{
		BaseURL:              cfg.BaseURL,
		BearerToken:          cfg.BearerToken,
		AuthProvider:         cfg.AuthProvider,
		Timeout:              cfg.Timeout,
		RebuildProbeInterval: cfg.RebuildProbeInterval,
		RetryMax:             cfg.RetryMax,
		RetryBackoff:         cfg.RetryBackoff,
		RetryJitter:          cfg.RetryJitter,
		MaxIdleConns:         cfg.MaxIdleConns,
		MaxIdlePerHost:       cfg.MaxIdlePerHost,
		TLSHandshake:         cfg.TLSHandshake,
		DisableKeepAlive:     cfg.DisableKeepAlive,
		HTTPClient:           cfg.HTTPClient,
		Logger:               cfg.Logger,
		Hooks:                clinicAPIHooks(cfg.Hooks),
	}
}
func clinicAPIHooks(h model.Hooks) apitypes.Hooks {
	return apitypes.Hooks{
		OnRequestStart: func(info apitypes.RequestInfo) {
			if h.OnRequestStart != nil {
				h.OnRequestStart(model.RequestInfo{
					Endpoint:  info.Endpoint,
					Method:    info.Method,
					Attempt:   info.Attempt,
					OrgType:   info.OrgType,
					OrgID:     info.OrgID,
					ClusterID: info.ClusterID,
					ItemID:    info.ItemID,
				})
			}
		},
		OnRequestDone: func(result apitypes.RequestResult) {
			if h.OnRequestDone != nil {
				h.OnRequestDone(model.RequestResult{
					RequestInfo: model.RequestInfo{
						Endpoint:  result.Endpoint,
						Method:    result.Method,
						Attempt:   result.Attempt,
						OrgType:   result.OrgType,
						OrgID:     result.OrgID,
						ClusterID: result.ClusterID,
						ItemID:    result.ItemID,
					},
					StatusCode:    result.StatusCode,
					Duration:      result.Duration,
					ResponseBytes: result.ResponseBytes,
				})
			}
		},
		OnRetry: func(retry apitypes.RequestRetry) {
			if h.OnRetry != nil {
				h.OnRetry(model.RequestRetry{
					RequestResult: model.RequestResult{
						RequestInfo: model.RequestInfo{
							Endpoint:  retry.Endpoint,
							Method:    retry.Method,
							Attempt:   retry.Attempt,
							OrgType:   retry.OrgType,
							OrgID:     retry.OrgID,
							ClusterID: retry.ClusterID,
							ItemID:    retry.ItemID,
						},
						StatusCode:    retry.StatusCode,
						Duration:      retry.Duration,
						ResponseBytes: retry.ResponseBytes,
					},
					ErrorClass: model.ErrorClass(retry.ErrorClass),
					Retryable:  retry.Retryable,
					Err:        mapAPIError(retry.Err),
				})
			}
		},
		OnError: func(failure apitypes.RequestFailure) {
			if h.OnError != nil {
				h.OnError(model.RequestFailure{
					RequestResult: model.RequestResult{
						RequestInfo: model.RequestInfo{
							Endpoint:  failure.Endpoint,
							Method:    failure.Method,
							Attempt:   failure.Attempt,
							OrgType:   failure.OrgType,
							OrgID:     failure.OrgID,
							ClusterID: failure.ClusterID,
							ItemID:    failure.ItemID,
						},
						StatusCode:    failure.StatusCode,
						Duration:      failure.Duration,
						ResponseBytes: failure.ResponseBytes,
					},
					ErrorClass: model.ErrorClass(failure.ErrorClass),
					Retryable:  failure.Retryable,
					Err:        mapAPIError(failure.Err),
				})
			}
		},
	}
}
func mapAPIError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *apitypes.Error
	if !errors.As(err, &apiErr) || apiErr == nil {
		return err
	}
	return &Error{
		Class:      ErrorClass(apiErr.Class),
		Retryable:  apiErr.Retryable,
		StatusCode: apiErr.StatusCode,
		Endpoint:   apiErr.Endpoint,
		Message:    apiErr.Message,
		Cause:      apiErr.Cause,
	}
}
func downloadedArtifactFromCloud(artifact apitypes.CloudDownloadedArtifact) DownloadedArtifact {
	return DownloadedArtifact{
		Filename:    artifact.Filename,
		ContentType: artifact.ContentType,
		Bytes:       append([]byte(nil), artifact.Bytes...),
	}
}
func rewriteClusterIDInPromQL(query, childClusterID, parentClusterID string) (string, bool) {
	childClusterID = strings.TrimSpace(childClusterID)
	parentClusterID = strings.TrimSpace(parentClusterID)
	if childClusterID == "" || parentClusterID == "" || !strings.Contains(query, childClusterID) {
		return "", false
	}
	return strings.ReplaceAll(query, childClusterID, parentClusterID), true
}
func rewriteClusterIDMatchers(matches []string, childClusterID, parentClusterID string) ([]string, bool) {
	childClusterID = strings.TrimSpace(childClusterID)
	parentClusterID = strings.TrimSpace(parentClusterID)
	if childClusterID == "" || parentClusterID == "" {
		return nil, false
	}
	rewritten := append([]string(nil), matches...)
	changed := false
	for i, item := range rewritten {
		if strings.Contains(item, childClusterID) {
			rewritten[i] = strings.ReplaceAll(item, childClusterID, parentClusterID)
			changed = true
		}
	}
	return rewritten, changed
}
