package cli

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	clinicapi "github.com/AricSu/tidb-clinic-client"
)

const (
	defaultMetricsQuery = "histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le, instance))"
	defaultStep         = "1m"
	defaultRangeWindow  = 10 * time.Minute
)

type cliConfig struct {
	BaseURL              string
	APIKey               string
	OrgID                string
	ClusterID            string
	Query                string
	Start                int64
	End                  int64
	Step                 string
	RebuildProbeInterval time.Duration
}

func loadConfigFromEnv(lookup func(string) (string, bool), now func() time.Time) (cliConfig, error) {
	portalInput, portalSet, err := resolvePortalURLInput(lookup)
	if err != nil {
		return cliConfig{}, err
	}
	if !portalSet {
		return cliConfig{}, fmt.Errorf("CLINIC_PORTAL_URL is required")
	}
	baseURL, apiKey, clusterID, err := resolveClinicAccess(lookup, portalInput.info, portalSet, true)
	if err != nil {
		return cliConfig{}, err
	}
	end := now().Unix()
	start := end - int64(defaultRangeWindow/time.Second)
	if portalInput.rangeEnd > 0 {
		end = portalInput.rangeEnd
	}
	if portalInput.rangeStart > 0 {
		start = portalInput.rangeStart
	} else if portalInput.rangeEnd > 0 {
		start = end - int64(defaultRangeWindow/time.Second)
	}
	if end < start {
		return cliConfig{}, fmt.Errorf("CLINIC_PORTAL_URL query parameter to must be greater than or equal to from")
	}
	rebuildProbeInterval := time.Duration(0)
	if raw, ok := optionalEnv(lookup, "CLINIC_REBUILD_PROBE_INTERVAL"); ok {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			return cliConfig{}, fmt.Errorf("CLINIC_REBUILD_PROBE_INTERVAL must be a positive duration")
		}
		rebuildProbeInterval = parsed
	}
	query, ok := optionalEnv(lookup, "CLINIC_METRICS_QUERY")
	if !ok {
		query = defaultMetricsQuery
	}
	step, ok := optionalEnv(lookup, "CLINIC_RANGE_STEP")
	if !ok {
		step = defaultStep
	}
	return cliConfig{
		BaseURL:              baseURL,
		APIKey:               apiKey,
		OrgID:                portalInput.orgID,
		ClusterID:            clusterID,
		Query:                query,
		Start:                start,
		End:                  end,
		Step:                 step,
		RebuildProbeInterval: rebuildProbeInterval,
	}, nil
}

type portalInput struct {
	info       clinicapi.PortalURLInfo
	orgID      string
	rangeStart int64
	rangeEnd   int64
}

func resolveClinicAccess(lookup func(string) (string, bool), portalInfo clinicapi.PortalURLInfo, portalSet bool, requireCluster bool) (baseURL, apiKey, clusterID string, err error) {
	baseURL, err = resolveClinicBaseURL(lookup, portalInfo, portalSet)
	if err != nil {
		return "", "", "", err
	}
	apiKey, err = resolveClinicAPIKey(lookup, baseURL)
	if err != nil {
		return "", "", "", err
	}
	clusterID, err = resolveClinicClusterID(lookup, portalInfo, portalSet, requireCluster)
	if err != nil {
		return "", "", "", err
	}
	return baseURL, apiKey, clusterID, nil
}

func resolvePortalURLInput(lookup func(string) (string, bool)) (portalInput, bool, error) {
	raw, ok := optionalEnv(lookup, "CLINIC_PORTAL_URL")
	if !ok {
		return portalInput{}, false, nil
	}
	info, err := clinicapi.ParsePortalURL(raw)
	if err != nil {
		return portalInput{}, false, fmt.Errorf("CLINIC_PORTAL_URL is invalid: %w", err)
	}
	rangeStart, rangeEnd, err := parsePortalURLRange(raw)
	if err != nil {
		return portalInput{}, false, fmt.Errorf("CLINIC_PORTAL_URL is invalid: %w", err)
	}
	orgID, err := parsePortalURLOrgID(raw)
	if err != nil {
		return portalInput{}, false, fmt.Errorf("CLINIC_PORTAL_URL is invalid: %w", err)
	}
	if rangeEnd > 0 && rangeStart > 0 && rangeEnd < rangeStart {
		return portalInput{}, false, fmt.Errorf("CLINIC_PORTAL_URL is invalid: query parameter to must be greater than or equal to from")
	}
	return portalInput{
		info:       info,
		orgID:      orgID,
		rangeStart: rangeStart,
		rangeEnd:   rangeEnd,
	}, true, nil
}

func resolveClinicBaseURL(lookup func(string) (string, bool), portalInfo clinicapi.PortalURLInfo, portalSet bool) (string, error) {
	if portalSet && strings.TrimSpace(portalInfo.BaseURL) != "" {
		return strings.TrimSpace(portalInfo.BaseURL), nil
	}
	return "", fmt.Errorf("CLINIC_PORTAL_URL is required")
}

func resolveClinicAPIKey(lookup func(string) (string, bool), baseURL string) (string, error) {
	if clinicBaseURLIsCN(baseURL) {
		if value, ok := optionalEnv(lookup, "CLINIC_CN_API_KEY"); ok {
			return value, nil
		}
		return "", fmt.Errorf("CLINIC_CN_API_KEY is required")
	}
	return requiredEnv(lookup, "CLINIC_API_KEY")
}

func resolveClinicClusterID(lookup func(string) (string, bool), portalInfo clinicapi.PortalURLInfo, portalSet bool, requireCluster bool) (string, error) {
	if portalSet && strings.TrimSpace(portalInfo.ClusterID) != "" {
		return strings.TrimSpace(portalInfo.ClusterID), nil
	}
	if !requireCluster {
		return "", nil
	}
	return "", fmt.Errorf("CLINIC_PORTAL_URL is required")
}

func parsePortalURLRange(raw string) (int64, int64, error) {
	route, err := parsePortalURLRoute(raw)
	if err != nil {
		return 0, 0, err
	}
	query := route.Query()
	rangeStart, err := parseOptionalInt64Query(query, "from")
	if err != nil {
		return 0, 0, err
	}
	rangeEnd, err := parseOptionalInt64Query(query, "to")
	if err != nil {
		return 0, 0, err
	}
	return rangeStart, rangeEnd, nil
}

func parsePortalURLOrgID(raw string) (string, error) {
	route, err := parsePortalURLRoute(raw)
	if err != nil {
		return "", err
	}
	segments := strings.Split(strings.Trim(route.Path, "/"), "/")
	for i := 0; i < len(segments)-1; i++ {
		if strings.TrimSpace(segments[i]) != "orgs" {
			continue
		}
		value, err := url.PathUnescape(strings.TrimSpace(segments[i+1]))
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("portal URL must include /orgs/{orgID}")
}

func parsePortalURLRoute(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if fragment := strings.TrimSpace(parsed.Fragment); fragment != "" {
		route, err := url.Parse(strings.TrimPrefix(fragment, "#"))
		if err == nil && strings.TrimSpace(route.Path) != "" {
			return route, nil
		}
	}
	return parsed, nil
}

func parseOptionalInt64Query(query url.Values, key string) (int64, error) {
	raw := strings.TrimSpace(query.Get(key))
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

func clinicBaseURLIsCN(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	return strings.HasSuffix(host, ".cn")
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
