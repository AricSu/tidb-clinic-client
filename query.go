package clinicapi

import (
	"github.com/AricSu/tidb-clinic-client/internal/clinic"
	"github.com/AricSu/tidb-clinic-client/internal/model"
)

type (
	TimeSeriesQuery              = clinic.TimeSeriesQuery
	MetricsCompileQuery          = model.MetricsCompileQuery
	LogRangeQuery                = model.LogRangeQuery
	SlowQueryQuery               = clinic.SlowQueryQuery
	SlowQuerySamplesQuery        = clinic.SlowQuerySamplesQuery
	LogLabelsQuery               = model.LogLabelsQuery
	LogLabelValuesQuery          = model.LogLabelValuesQuery
	ProfileFetchRequest          = model.ProfileFetchRequest
	DiagnosticDownloadRequest    = model.DiagnosticDownloadRequest
	CollectedDataDownloadRequest = model.CollectedDataDownloadRequest
)
