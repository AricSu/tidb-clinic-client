package clinicapi

import (
	"github.com/AricSu/tidb-clinic-client/internal/clinic"
	"github.com/AricSu/tidb-clinic-client/internal/model"
)

type (
	QueryMetadata            = clinic.QueryMetadata
	RetainedDataRef          = clinic.RetainedDataRef
	SeriesKind               = clinic.SeriesKind
	SeriesPoint              = clinic.SeriesPoint
	Series                   = clinic.Series
	SeriesResult             = clinic.SeriesResult
	StreamValue              = clinic.StreamValue
	Stream                   = clinic.Stream
	StreamResult             = clinic.StreamResult
	TableResult              = clinic.TableResult
	ListResult               = clinic.ListResult
	ObjectResult             = clinic.ObjectResult
	BlobResult               = clinic.BlobResult
	MetricQueryRangeResult   = clinic.MetricQueryRangeResult
	MetricQueryInstantResult = clinic.MetricQueryInstantResult
	MetricQuerySeriesResult  = clinic.MetricQuerySeriesResult
	LogQueryResult           = clinic.LogQueryResult
	LogLabelsResult          = clinic.LogLabelsResult
	LogSearchResult          = clinic.LogSearchResult
	AnalyticalResult         = clinic.AnalyticalResult
	SchemaResult             = clinic.SchemaResult
	TopSQLSummaryResult      = clinic.TopSQLSummaryResult
	SlowQuerySummaryResult   = clinic.SlowQuerySummaryResult
	SlowQuerySamplesResult   = clinic.SlowQuerySamplesResult
	SlowQueryDetail          = clinic.SlowQueryDetail
	SlowQueryRecordsResult   = clinic.SlowQueryRecordsResult
	ConfigResult             = clinic.ConfigResult
	ProfileGroupsResult      = clinic.ProfileGroupsResult
	ProfileGroupDetail       = clinic.ProfileGroupDetail
	DiagnosticListResult     = clinic.DiagnosticListResult
	DownloadedArtifact       = clinic.DownloadedArtifact
)

const (
	SeriesKindRange   = model.SeriesKindRange
	SeriesKindInstant = model.SeriesKindInstant
	SeriesKindSet     = model.SeriesKindSet
)
