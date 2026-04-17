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
	VerboseRequestLogs   bool
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
		Timeout:              20 * time.Second,
		RebuildProbeInterval: 10 * time.Second,
		RetryMax:             2,
		RetryBackoff:         250 * time.Millisecond,
		RetryJitter:          250 * time.Millisecond,
		MaxIdleConns:         64,
		MaxIdlePerHost:       16,
		TLSHandshake:         10 * time.Second,
	}
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
	if c.Timeout <= 0 {
		return errors.New("clinic api timeout must be positive")
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

type CapabilityName string

const (
	CapabilityClusterDetail   CapabilityName = "cluster_detail"
	CapabilityTopology        CapabilityName = "topology"
	CapabilityEvents          CapabilityName = "events"
	CapabilityMetrics         CapabilityName = "metrics"
	CapabilityLogs            CapabilityName = "logs"
	CapabilitySQLQuery        CapabilityName = "sql_query"
	CapabilitySchema          CapabilityName = "schema"
	CapabilityTopSQL          CapabilityName = "topsql"
	CapabilitySlowQuery       CapabilityName = "slow_query"
	CapabilitySQLStatements   CapabilityName = "sql_statements"
	CapabilityConfigs         CapabilityName = "configs"
	CapabilityProfiling       CapabilityName = "profiling"
	CapabilityDiagnosticFiles CapabilityName = "diagnostic_files"
)

type CapabilityScope string

const (
	CapabilityScopeCluster CapabilityScope = "cluster"
)

type CapabilityStability string

const (
	CapabilityStabilityStable      CapabilityStability = "stable"
	CapabilityStabilityPlaceholder CapabilityStability = "placeholder"
)

type CapabilityDescriptor struct {
	Name                 CapabilityName
	Available            bool
	Reason               string
	Scope                CapabilityScope
	Stability            CapabilityStability
	RequiresParentTarget bool
	RequiresLiveCluster  bool
	TierConstraints      []string
}
type ClusterRecord struct {
	ClusterID    string
	Name         string
	OrgID        string
	ClusterType  string
	TenantID     string
	ProjectID    string
	Provider     string
	Region       string
	DeployType   string
	DeployTypeV2 string
	ParentID     string
	Status       string
	Deleted      bool
}
type ClusterMetadata struct {
	Platform     TargetPlatform
	ClusterID    string
	OrgID        string
	ClusterType  string
	Provider     string
	Region       string
	DeployType   string
	DeployTypeV2 string
	ParentID     string
	Status       string
	Deleted      bool
}
type ClusterCapabilities struct {
	Cluster      ClusterMetadata
	Capabilities []CapabilityDescriptor
}

func (c ClusterCapabilities) Lookup(name CapabilityName) (CapabilityDescriptor, bool) {
	for _, item := range c.Capabilities {
		if item.Name == name {
			return item, true
		}
	}
	return CapabilityDescriptor{}, false
}

type QueryMetadata struct {
	RowCount      int64
	BytesScanned  int64
	ExecutionTime string
	QueryID       string
	Engine        string
	Vendor        string
	Region        string
	Partial       bool
	Warnings      []string
	Raw           map[string]any
}
type RetainedDataRef struct {
	ItemID    string
	SourceRef string
}
type ClusterSearchQuery struct {
	Query       string
	ClusterID   string
	ShowDeleted bool
	Limit       int
	Page        int
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

type LogQuery struct {
	Query     string
	Time      int64
	Limit     int
	Direction string
}
type LogRangeQuery struct {
	Query     string
	Start     int64
	End       int64
	Limit     int
	Direction string
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
type LogSearchQuery struct {
	StartTime int64
	EndTime   int64
	Pattern   string
	Limit     int
}
type SQLQuery struct {
	SQL     string
	Timeout int
}
type SchemaQuery struct {
	Tables []string
}
type TopSQLSummaryQuery struct {
	Component string
	Instance  string
	Start     string
	End       string
	Top       int
	Window    string
	GroupBy   string
}
type TopSlowQueriesQuery struct {
	Start   string
	Hours   int
	OrderBy string
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
type SlowQueryDetailQuery struct {
	ID           string
	Start        string
	End          string
	Digest       string
	ConnectionID string
	Timestamp    string
}
type SlowQueryRecordsQuery struct {
	StartTime int64
	EndTime   int64
	OrderBy   string
	Desc      bool
	Limit     int
}
type SQLStatementsQuery struct {
	SQL     string
	Timeout int
	Start   string
	End     string
}
type ProfileActionTokenRequest struct {
	Timestamp   int64
	ProfileType string
	Component   string
	Address     string
	DataFormat  string
}
type ProfileDownloadRequest struct {
	Token string
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
type ConfigQuery struct{}
type SeriesKind string

const (
	SeriesKindRange   SeriesKind = "range"
	SeriesKindInstant SeriesKind = "instant"
	SeriesKindSet     SeriesKind = "set"
)

type SeriesPoint struct {
	Timestamp int64
	Value     string
}
type Series struct {
	Labels map[string]string
	Values []SeriesPoint
}
type SeriesResult struct {
	Kind      SeriesKind
	IsPartial bool
	Series    []Series
	Metadata  QueryMetadata
}
type StreamValue struct {
	Timestamp string
	Line      string
}
type Stream struct {
	Labels map[string]string
	Values []StreamValue
}
type StreamResult struct {
	Status   string
	Streams  []Stream
	Metadata QueryMetadata
}
type TableResult struct {
	Columns  []string
	Rows     []map[string]any
	Metadata QueryMetadata
}
type ListResult struct {
	Total    int
	Items    []map[string]any
	Metadata QueryMetadata
}
type ObjectResult struct {
	Fields   map[string]any
	Metadata QueryMetadata
}
type BlobResult struct {
	Filename    string
	ContentType string
	Bytes       []byte
}
type DownloadedArtifact = BlobResult
