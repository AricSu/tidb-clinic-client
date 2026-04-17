package clinicapi

import (
	"github.com/AricSu/tidb-clinic-client/internal/clinic"
	"github.com/AricSu/tidb-clinic-client/internal/model"
)

type (
	TimeSeriesQuery           = clinic.TimeSeriesQuery
	LogQuery                  = model.LogQuery
	LogRangeQuery             = model.LogRangeQuery
	LogLabelsQuery            = model.LogLabelsQuery
	LogLabelValuesQuery       = model.LogLabelValuesQuery
	LogSearchQuery            = model.LogSearchQuery
	SQLQuery                  = model.SQLQuery
	SchemaQuery               = model.SchemaQuery
	TopSQLSummaryQuery        = model.TopSQLSummaryQuery
	TopSlowQueriesQuery       = model.TopSlowQueriesQuery
	SlowQuerySamplesQuery     = clinic.SlowQuerySamplesQuery
	SlowQueryDetailQuery      = clinic.SlowQueryDetailQuery
	SlowQueryRecordsQuery     = clinic.SlowQueryRecordsQuery
	SQLStatementsQuery        = model.SQLStatementsQuery
	ProfileActionTokenRequest = model.ProfileActionTokenRequest
	ProfileDownloadRequest    = model.ProfileDownloadRequest
	ProfileFetchRequest       = model.ProfileFetchRequest
	DiagnosticDownloadRequest = model.DiagnosticDownloadRequest
	ConfigQuery               = model.ConfigQuery
)
