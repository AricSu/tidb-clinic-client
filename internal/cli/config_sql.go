package cli

import (
	"errors"
	"strconv"
	"time"
)

type cloudTopSQLConfig struct {
	Base      cliConfig
	Component string
	Instance  string
	Start     string
	End       string
	Top       int
	Window    string
	GroupBy   string
}
type cloudTopSlowQueriesConfig struct {
	Base    cliConfig
	Start   string
	Hours   int
	OrderBy string
	Limit   int
}
type cloudSlowQueryListConfig struct {
	Base    cliConfig
	Digest  string
	Start   string
	End     string
	OrderBy string
	Limit   int
	Desc    bool
	Fields  []string
}
type cloudSlowQueryDetailConfig struct {
	ID           string
	Base         cliConfig
	Digest       string
	ConnectionID string
	Timestamp    string
}
type cloudDataProxyQueryConfig struct {
	Base    cliConfig
	SQL     string
	Timeout int
}
type cloudDataProxySchemaConfig struct {
	Base   cliConfig
	Tables []string
}

func loadCloudTopSQLConfig(lookup func(string) (string, bool), now func() time.Time) (cloudTopSQLConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudTopSQLConfig{}, err
	}
	component, err := requiredEnv(lookup, "CLINIC_COMPONENT")
	if err != nil {
		return cloudTopSQLConfig{}, err
	}
	instance, err := requiredEnv(lookup, "CLINIC_INSTANCE")
	if err != nil {
		return cloudTopSQLConfig{}, err
	}
	top, err := optionalPositiveIntEnv(lookup, "CLINIC_LIMIT")
	if err != nil {
		return cloudTopSQLConfig{}, err
	}
	window, _ := optionalEnv(lookup, "CLINIC_TOPSQL_WINDOW")
	groupBy, _ := optionalEnv(lookup, "CLINIC_TOPSQL_GROUP_BY")
	return cloudTopSQLConfig{
		Base:      base,
		Component: component,
		Instance:  instance,
		Start:     strconv.FormatInt(base.Start, 10),
		End:       strconv.FormatInt(base.End, 10),
		Top:       top,
		Window:    window,
		GroupBy:   groupBy,
	}, nil
}
func loadCloudTopSlowQueriesConfig(lookup func(string) (string, bool), now func() time.Time) (cloudTopSlowQueriesConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudTopSlowQueriesConfig{}, err
	}
	orderBy, _ := optionalEnv(lookup, "CLINIC_SLOWQUERY_ORDER_BY")
	limit, err := optionalPositiveIntEnv(lookup, "CLINIC_LIMIT")
	if err != nil {
		return cloudTopSlowQueriesConfig{}, err
	}
	return cloudTopSlowQueriesConfig{
		Base:    base,
		Start:   strconv.FormatInt(base.Start, 10),
		Hours:   rangeHours(base.Start, base.End),
		OrderBy: orderBy,
		Limit:   limit,
	}, nil
}
func loadCloudSlowQueryListConfig(lookup func(string) (string, bool), now func() time.Time) (cloudSlowQueryListConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudSlowQueryListConfig{}, err
	}
	digest, err := requiredEnv(lookup, "CLINIC_SLOWQUERY_DIGEST")
	if err != nil {
		return cloudSlowQueryListConfig{}, err
	}
	orderBy, _ := optionalEnv(lookup, "CLINIC_SLOWQUERY_ORDER_BY")
	limit, err := optionalPositiveIntEnv(lookup, "CLINIC_LIMIT")
	if err != nil {
		return cloudSlowQueryListConfig{}, err
	}
	desc := false
	if raw, ok := optionalEnv(lookup, "CLINIC_DESC"); ok {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return cloudSlowQueryListConfig{}, &parseEnvError{key: "CLINIC_DESC", message: "must be true or false"}
		}
		desc = parsed
	}
	fields := defaultCloudFields()
	if raw, ok := optionalEnv(lookup, "CLINIC_SLOWQUERY_FIELDS"); ok {
		fields = splitCSV(raw)
	}
	return cloudSlowQueryListConfig{
		Base:    base,
		Digest:  digest,
		Start:   strconv.FormatInt(base.Start, 10),
		End:     strconv.FormatInt(base.End, 10),
		OrderBy: orderBy,
		Limit:   limit,
		Desc:    desc,
		Fields:  fields,
	}, nil
}
func loadCloudSlowQueryDetailConfig(lookup func(string) (string, bool), now func() time.Time) (cloudSlowQueryDetailConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudSlowQueryDetailConfig{}, err
	}
	id, _ := optionalEnv(lookup, "CLINIC_SLOWQUERY_ID")
	digest, _ := optionalEnv(lookup, "CLINIC_SLOWQUERY_DIGEST")
	connectionID, _ := optionalEnv(lookup, "CLINIC_SLOWQUERY_CONNECTION_ID")
	timestamp, _ := optionalEnv(lookup, "CLINIC_SLOWQUERY_TIMESTAMP")
	switch {
	case id != "":
		// TiUP slow-query detail can resolve the collected item automatically from the shared range.
		// Ignore extra compatibility fields so stale shell exports do not shadow detail output.
		digest = ""
		connectionID = ""
		timestamp = ""
	case digest != "" || connectionID != "" || timestamp != "":
		if digest == "" {
			return cloudSlowQueryDetailConfig{}, errors.New("CLINIC_SLOWQUERY_DIGEST is required")
		}
		if connectionID == "" {
			return cloudSlowQueryDetailConfig{}, errors.New("CLINIC_SLOWQUERY_CONNECTION_ID is required")
		}
		if timestamp == "" {
			return cloudSlowQueryDetailConfig{}, errors.New("CLINIC_SLOWQUERY_TIMESTAMP is required")
		}
	default:
		return cloudSlowQueryDetailConfig{}, errors.New("slow query detail requires CLINIC_SLOWQUERY_ID, or CLINIC_SLOWQUERY_DIGEST + CLINIC_SLOWQUERY_CONNECTION_ID + CLINIC_SLOWQUERY_TIMESTAMP")
	}
	return cloudSlowQueryDetailConfig{
		ID:           id,
		Base:         base,
		Digest:       digest,
		ConnectionID: connectionID,
		Timestamp:    timestamp,
	}, nil
}
func loadCloudDataProxyQueryConfig(lookup func(string) (string, bool), now func() time.Time) (cloudDataProxyQueryConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudDataProxyQueryConfig{}, err
	}
	sql, err := requiredEnv(lookup, "CLINIC_DATA_PROXY_SQL")
	if err != nil {
		return cloudDataProxyQueryConfig{}, err
	}
	timeout, err := optionalPositiveIntEnv(lookup, "CLINIC_DATA_PROXY_TIMEOUT")
	if err != nil {
		return cloudDataProxyQueryConfig{}, err
	}
	return cloudDataProxyQueryConfig{Base: base, SQL: sql, Timeout: timeout}, nil
}
func loadCloudDataProxySchemaConfig(lookup func(string) (string, bool), now func() time.Time) (cloudDataProxySchemaConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudDataProxySchemaConfig{}, err
	}
	tables, err := requiredEnv(lookup, "CLINIC_DATA_PROXY_TABLES")
	if err != nil {
		return cloudDataProxySchemaConfig{}, err
	}
	return cloudDataProxySchemaConfig{Base: base, Tables: splitCSV(tables)}, nil
}
func rangeHours(start, end int64) int {
	if end <= start {
		return 1
	}
	seconds := end - start
	hours := int((seconds + 3599) / 3600)
	if hours < 1 {
		return 1
	}
	return hours
}
