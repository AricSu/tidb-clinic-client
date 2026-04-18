package clinicapi

import (
	"github.com/AricSu/tidb-clinic-client/internal/clinic"
	"github.com/AricSu/tidb-clinic-client/internal/model"
)

type (
	QueryMetadata                  = clinic.QueryMetadata
	RetainedDataRef                = clinic.RetainedDataRef
	CollectedDataItem              = clinic.CollectedDataItem
	SeriesKind                     = clinic.SeriesKind
	SeriesPoint                    = clinic.SeriesPoint
	Series                         = clinic.Series
	SeriesResult                   = clinic.SeriesResult
	StreamValue                    = clinic.StreamValue
	Stream                         = clinic.Stream
	StreamResult                   = clinic.StreamResult
	ListResult                     = clinic.ListResult
	BlobResult                     = clinic.BlobResult
	MetricQueryRangeResult         = clinic.MetricQueryRangeResult
	CompiledTimeseriesProblemRange = clinic.CompiledTimeseriesProblemRange
	CompiledTimeseriesEvent        = clinic.CompiledTimeseriesEvent
	CompiledTimeseriesDigest       = clinic.CompiledTimeseriesDigest
	LogQueryResult                 = clinic.LogQueryResult
	SlowQueryRecord                = clinic.SlowQueryRecord
	SlowQueryResult                = clinic.SlowQueryResult
	SlowQuerySamplesResult         = clinic.SlowQuerySamplesResult
	LogLabelsResult                = clinic.LogLabelsResult
	ProfileGroupsResult            = clinic.ProfileGroupsResult
	DiagnosticListResult           = clinic.DiagnosticListResult
	DownloadedArtifact             = clinic.DownloadedArtifact
)

const (
	SeriesKindRange = model.SeriesKindRange
)
