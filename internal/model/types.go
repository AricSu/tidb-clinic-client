package model

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AuthProvider interface {
	Apply(req *http.Request) error
}
type AuthProviderFunc func(req *http.Request) error

func (f AuthProviderFunc) Apply(req *http.Request) error {
	if f == nil {
		return nil
	}
	return f(req)
}

type BearerTokenAuthProvider struct {
	Token string
}

func (p BearerTokenAuthProvider) Apply(req *http.Request) error {
	if req == nil {
		return errors.New("request is nil")
	}
	token := strings.TrimSpace(p.Token)
	if token == "" {
		return errors.New("bearer token is empty")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}
func StaticBearerToken(token string) AuthProvider {
	return BearerTokenAuthProvider{Token: token}
}

type ErrorClass string

const (
	ErrInvalidRequest ErrorClass = "invalid_request"
	ErrUnsupported    ErrorClass = "unsupported"
	ErrAuth           ErrorClass = "auth"
	ErrNotFound       ErrorClass = "not_found"
	ErrNoData         ErrorClass = "no_data"
	ErrTimeout        ErrorClass = "timeout"
	ErrRateLimit      ErrorClass = "rate_limit"
	ErrDecode         ErrorClass = "decode"
	ErrBackend        ErrorClass = "backend"
	ErrTransient      ErrorClass = "transient"
)

type Error struct {
	Class      ErrorClass
	Retryable  bool
	StatusCode int
	Endpoint   string
	Message    string
	Cause      error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	switch {
	case e.Endpoint != "" && e.Message != "":
		return fmt.Sprintf("%s: %s", e.Endpoint, e.Message)
	case e.Message != "":
		return e.Message
	case e.Endpoint != "":
		return e.Endpoint
	default:
		return string(e.Class)
	}
}
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}
func IsRetryable(err error) bool {
	var clinicErr *Error
	if !errors.As(err, &clinicErr) || clinicErr == nil {
		return false
	}
	return clinicErr.Retryable
}
func ClassOf(err error) ErrorClass {
	var clinicErr *Error
	if !errors.As(err, &clinicErr) || clinicErr == nil {
		return ""
	}
	return clinicErr.Class
}

type Hooks struct {
	OnRequestStart func(RequestInfo)
	OnRequestDone  func(RequestResult)
	OnRetry        func(RequestRetry)
	OnError        func(RequestFailure)
}
type RequestInfo struct {
	Endpoint  string
	Method    string
	Attempt   int
	OrgType   string
	OrgID     string
	ClusterID string
	ItemID    string
}
type RequestResult struct {
	RequestInfo
	StatusCode    int
	Duration      time.Duration
	ResponseBytes int
}
type RequestRetry struct {
	RequestResult
	ErrorClass ErrorClass
	Retryable  bool
	Err        error
}
type RequestFailure struct {
	RequestResult
	ErrorClass ErrorClass
	Retryable  bool
	Err        error
}
type Config struct {
	BaseURL              string
	BearerToken          string
	AuthProvider         AuthProvider
	Timeout              time.Duration
	RebuildProbeInterval time.Duration
	RetryMax             int
	RetryBackoff         time.Duration
	RetryJitter          time.Duration
	MaxIdleConns         int
	MaxIdlePerHost       int
	TLSHandshake         time.Duration
	DisableKeepAlive     bool
	HTTPClient           *http.Client
	Logger               *log.Logger
	Hooks                Hooks
}

func DefaultConfig() Config {
	return Config{
		RebuildProbeInterval: 10 * time.Second,
		RetryMax:             2,
		RetryBackoff:         250 * time.Millisecond,
		RetryJitter:          250 * time.Millisecond,
		MaxIdleConns:         64,
		MaxIdlePerHost:       16,
		TLSHandshake:         10 * time.Second,
	}
}

func MergeConfig(cfg Config) Config {
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
	return merged
}

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
	if c.RebuildProbeInterval <= 0 {
		return errors.New("clinic api rebuild probe interval must be positive")
	}
	if c.RetryMax < 0 {
		return errors.New("clinic api retry max must be non-negative")
	}
	if c.RetryBackoff < 0 || c.RetryJitter < 0 {
		return errors.New("clinic api retry backoff and jitter must be non-negative")
	}
	return nil
}

type TargetPlatform string

const (
	TargetPlatformCloud       TargetPlatform = "cloud"
	TargetPlatformTiUPCluster TargetPlatform = "tiup-cluster"
)

type QueryMetadata struct {
	RowCount      int64          `json:"rowCount"`
	BytesScanned  int64          `json:"bytesScanned"`
	ExecutionTime string         `json:"executionTime"`
	QueryID       string         `json:"queryID"`
	Engine        string         `json:"engine"`
	Vendor        string         `json:"vendor"`
	Region        string         `json:"region"`
	Partial       bool           `json:"partial"`
	Warnings      []string       `json:"warnings,omitempty"`
	Raw           map[string]any `json:"raw,omitempty"`
}
type RetainedDataRef struct {
	ItemID    string
	SourceRef string
}
type CollectedDataItem struct {
	ItemID     string   `json:"itemID"`
	Filename   string   `json:"filename"`
	Collectors []string `json:"collectors,omitempty"`
	HaveLog    bool     `json:"haveLog"`
	HaveMetric bool     `json:"haveMetric"`
	HaveConfig bool     `json:"haveConfig"`
	StartTime  int64    `json:"startTime"`
	EndTime    int64    `json:"endTime"`
}
type TimeSeriesQuery struct {
	Query   string
	Match   []string
	Time    int64
	Start   int64
	End     int64
	Step    string
	Timeout string
}

type MetricsCompileQuery struct {
	Query            string   `json:"query"`
	Start            int64    `json:"start"`
	End              int64    `json:"end"`
	Step             string   `json:"step"`
	Timeout          string   `json:"timeout,omitempty"`
	MetricID         string   `json:"metricID,omitempty"`
	ExprDescription  string   `json:"exprDescription,omitempty"`
	LabelsOfInterest []string `json:"labelsOfInterest,omitempty"`
	SourceRef        string   `json:"sourceRef,omitempty"`
}

func (q TimeSeriesQuery) Clone() TimeSeriesQuery {
	q.Match = append([]string(nil), q.Match...)
	return q
}
func (q TimeSeriesQuery) ComparedTo(offsetSeconds int64) (TimeSeriesQuery, TimeSeriesQuery) {
	base := q.Clone()
	compare := q.Clone()
	compare.Time -= offsetSeconds
	compare.Start -= offsetSeconds
	compare.End -= offsetSeconds
	return base, compare
}

type LogRangeQuery struct {
	Query     string
	Start     int64
	End       int64
	Limit     int
	Direction string
}
type SlowQueryQuery struct {
	Start   int64
	End     int64
	OrderBy string
	Desc    bool
	Limit   int
}
type SlowQuerySamplesQuery struct {
	Digest  string
	Start   string
	End     string
	OrderBy string
	Limit   int
	Desc    bool
	Fields  []string
}
type LogLabelsQuery struct {
	Start int64
	End   int64
}
type LogLabelValuesQuery struct {
	LabelName string
	Start     int64
	End       int64
}
type ProfileFetchRequest struct {
	Timestamp   int64
	ProfileType string
	Component   string
	Address     string
	DataFormat  string
}
type DiagnosticDownloadRequest struct {
	Key string
}
type CollectedDataDownloadRequest struct {
	StartTime int64
	EndTime   int64
}
type SeriesKind string

const (
	SeriesKindRange SeriesKind = "range"
)

type SeriesPoint struct {
	Timestamp int64  `json:"timestamp"`
	Value     string `json:"value"`
}
type Series struct {
	Labels map[string]string `json:"labels,omitempty"`
	Values []SeriesPoint     `json:"values,omitempty"`
}
type SeriesResult struct {
	Kind      SeriesKind    `json:"kind"`
	IsPartial bool          `json:"isPartial"`
	Series    []Series      `json:"series,omitempty"`
	Metadata  QueryMetadata `json:"metadata,omitempty"`
}

type CompiledTimeseriesProblemRange struct {
	StartTSSecs int64 `json:"start_ts_secs"`
	EndTSSecs   int64 `json:"end_ts_secs"`
}

type CompiledTimeseriesEvent struct {
	Kind        string  `json:"kind,omitempty"`
	Score       float64 `json:"score,omitempty"`
	StartTSSecs int64   `json:"start_ts_secs"`
	EndTSSecs   int64   `json:"end_ts_secs"`
}

type CompiledTimeseriesDigest struct {
	MetricID     string                          `json:"metric_id,omitempty"`
	Scope        string                          `json:"scope,omitempty"`
	SubjectID    string                          `json:"subject_id,omitempty"`
	Summary      string                          `json:"summary,omitempty"`
	State        string                          `json:"state,omitempty"`
	Trend        string                          `json:"trend,omitempty"`
	ProblemRange *CompiledTimeseriesProblemRange `json:"problem_range,omitempty"`
	TopEvents    []CompiledTimeseriesEvent       `json:"top_events,omitempty"`
	SourceRefs   []string                        `json:"source_refs,omitempty"`
}
type StreamValue struct {
	Timestamp string `json:"timestamp"`
	Line      string `json:"line"`
}
type SlowQueryRecord struct {
	Digest     string   `json:"digest"`
	SQLText    string   `json:"sqlText"`
	QueryTime  float64  `json:"queryTime"`
	ExecCount  int64    `json:"execCount"`
	User       string   `json:"user"`
	DB         string   `json:"db"`
	TableNames []string `json:"tableNames,omitempty"`
	IndexNames []string `json:"indexNames,omitempty"`
	SourceRef  string   `json:"sourceRef"`
}
type SlowQueryResult struct {
	Total   int               `json:"total"`
	Records []SlowQueryRecord `json:"records,omitempty"`
}
type Stream struct {
	Labels map[string]string `json:"labels,omitempty"`
	Values []StreamValue     `json:"values,omitempty"`
}
type StreamResult struct {
	Status   string        `json:"status"`
	Streams  []Stream      `json:"streams,omitempty"`
	Metadata QueryMetadata `json:"metadata,omitempty"`
}
type ListResult struct {
	Total    int              `json:"total"`
	Items    []map[string]any `json:"items,omitempty"`
	Metadata QueryMetadata    `json:"metadata,omitempty"`
}
type BlobResult struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Bytes       []byte `json:"bytes,omitempty"`
}
type DownloadedArtifact = BlobResult
