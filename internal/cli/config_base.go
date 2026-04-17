package cli

import (
	"context"
	"fmt"
	clinicapi "github.com/AricSu/tidb-clinic-client"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL      = "https://clinic.pingcap.com"
	defaultMetricsQuery = "sum(tidb_server_connections)"
	defaultStep         = "1m"
	defaultTimeout      = 20 * time.Second
	defaultRangeWindow  = 10 * time.Minute
)

type cliConfig struct {
	BaseURL              string
	APIKey               string
	ClusterID            string
	Query                string
	Match                []string
	Time                 int64
	Start                int64
	End                  int64
	Step                 string
	Timeout              time.Duration
	RebuildProbeInterval time.Duration
	VerboseLogs          bool
}

func loadConfigFromEnv(lookup func(string) (string, bool), now func() time.Time) (cliConfig, error) {
	baseURL, ok := optionalEnv(lookup, "CLINIC_API_BASE_URL")
	if !ok {
		baseURL = defaultBaseURL
	}
	apiKey, err := requiredEnv(lookup, "CLINIC_API_KEY")
	if err != nil {
		return cliConfig{}, err
	}
	clusterID, err := requiredEnv(lookup, "CLINIC_CLUSTER_ID")
	if err != nil {
		return cliConfig{}, err
	}
	end := now().Unix()
	if raw, ok := optionalEnv(lookup, "CLINIC_RANGE_END"); ok {
		end, err = parseInt64Env("CLINIC_RANGE_END", raw)
		if err != nil {
			return cliConfig{}, err
		}
	}
	start := end - int64(defaultRangeWindow/time.Second)
	if raw, ok := optionalEnv(lookup, "CLINIC_RANGE_START"); ok {
		start, err = parseInt64Env("CLINIC_RANGE_START", raw)
		if err != nil {
			return cliConfig{}, err
		}
	}
	if end < start {
		return cliConfig{}, fmt.Errorf("CLINIC_RANGE_END must be greater than or equal to CLINIC_RANGE_START")
	}
	timeout := defaultTimeout
	if raw, ok := optionalEnv(lookup, "CLINIC_TIMEOUT_SEC"); ok {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return cliConfig{}, fmt.Errorf("CLINIC_TIMEOUT_SEC must be a positive integer")
		}
		timeout = time.Duration(seconds) * time.Second
	}
	rebuildProbeInterval := time.Duration(0)
	if raw, ok := optionalEnv(lookup, "CLINIC_REBUILD_PROBE_INTERVAL"); ok {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			return cliConfig{}, fmt.Errorf("CLINIC_REBUILD_PROBE_INTERVAL must be a positive duration")
		}
		rebuildProbeInterval = parsed
	}
	verboseLogs := false
	if raw, ok := optionalEnv(lookup, "CLINIC_VERBOSE_LOGS"); ok {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return cliConfig{}, fmt.Errorf("CLINIC_VERBOSE_LOGS must be true or false")
		}
		verboseLogs = parsed
	}
	query, ok := optionalEnv(lookup, "CLINIC_METRICS_QUERY")
	if !ok {
		query = defaultMetricsQuery
	}
	matches, err := optionalMetricMatchers(lookup, "CLINIC_METRICS_MATCH")
	if err != nil {
		return cliConfig{}, err
	}
	queryTime := end
	if raw, ok := optionalEnv(lookup, "CLINIC_QUERY_TIME"); ok {
		queryTime, err = parseInt64Env("CLINIC_QUERY_TIME", raw)
		if err != nil {
			return cliConfig{}, err
		}
	}
	step, ok := optionalEnv(lookup, "CLINIC_RANGE_STEP")
	if !ok {
		step = defaultStep
	}
	return cliConfig{
		BaseURL:              baseURL,
		APIKey:               apiKey,
		ClusterID:            clusterID,
		Query:                query,
		Match:                matches,
		Time:                 queryTime,
		Start:                start,
		End:                  end,
		Step:                 step,
		Timeout:              timeout,
		RebuildProbeInterval: rebuildProbeInterval,
		VerboseLogs:          verboseLogs,
	}, nil
}
func (cfg cliConfig) resolveHandle(ctx context.Context, client *clinicapi.Client) (*clinicapi.ClusterHandle, error) {
	return client.Clusters.Resolve(ctx, cfg.ClusterID)
}
func requiredEnv(lookup func(string) (string, bool), key string) (string, error) {
	value, ok := optionalEnv(lookup, key)
	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}
func optionalEnv(lookup func(string) (string, bool), key string) (string, bool) {
	value, ok := lookup(key)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}
func parseInt64Env(key, value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", key)
	}
	return parsed, nil
}
func optionalMetricMatchers(lookup func(string) (string, bool), key string) ([]string, error) {
	raw, ok := optionalEnv(lookup, key)
	if !ok {
		return nil, nil
	}
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s must contain at least one non-empty selector", key)
	}
	return out, nil
}

type parseEnvError struct {
	key     string
	message string
}

func (e *parseEnvError) Error() string {
	return e.key + " " + e.message
}
func optionalPositiveIntEnv(lookup func(string) (string, bool), key string) (int, error) {
	raw, ok := optionalEnv(lookup, key)
	if !ok {
		return 0, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return 0, &parseEnvError{key: key, message: "must be a positive integer"}
	}
	return parsed, nil
}
func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
func defaultCloudFields() []string {
	return []string{"query", "timestamp", "query_time", "memory_max", "request_count", "connection_id"}
}
func toString(v any) string {
	return strings.TrimSpace(strings.ReplaceAll(fmt.Sprint(v), "\n", " "))
}
