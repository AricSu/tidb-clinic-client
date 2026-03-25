package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	clinicapi "github.com/aric/tidb-clinic-client"
)

const (
	defaultBaseURL      = "https://clinic.pingcap.com"
	defaultMetricsQuery = "sum(tidb_server_connections)"
	defaultStep         = "1m"
	defaultTimeout      = 20 * time.Second
	defaultRangeWindow  = 10 * time.Minute
)

type cliConfig struct {
	BaseURL string
	APIKey  string
	Context clinicapi.RequestContext
	Query   string
	Start   int64
	End     int64
	Step    string
	Timeout time.Duration
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
	orgType, ok := optionalEnv(lookup, "CLINIC_ORG_TYPE")
	if !ok {
		orgType = "cloud"
	}
	orgID, _ := optionalEnv(lookup, "CLINIC_ORG_ID")
	if orgID == "" && !strings.EqualFold(orgType, "cloud") {
		return cliConfig{}, errors.New("CLINIC_ORG_ID is required for non-cloud CLI runs")
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
		return cliConfig{}, errors.New("CLINIC_RANGE_END must be greater than or equal to CLINIC_RANGE_START")
	}

	timeout := defaultTimeout
	if raw, ok := optionalEnv(lookup, "CLINIC_TIMEOUT_SEC"); ok {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return cliConfig{}, fmt.Errorf("CLINIC_TIMEOUT_SEC must be a positive integer")
		}
		timeout = time.Duration(seconds) * time.Second
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
		BaseURL: baseURL,
		APIKey:  apiKey,
		Context: clinicapi.RequestContext{
			OrgType:   orgType,
			OrgID:     orgID,
			ClusterID: clusterID,
		},
		Query:   query,
		Start:   start,
		End:     end,
		Step:    step,
		Timeout: timeout,
	}, nil
}

func loadOPConfigFromEnv(lookup func(string) (string, bool), now func() time.Time) (cliConfig, error) {
	cfg, err := loadConfigFromEnv(withDefaultEnvValue(lookup, "CLINIC_ORG_TYPE", "op"), now)
	if err != nil {
		return cliConfig{}, err
	}
	if !strings.EqualFold(strings.TrimSpace(cfg.Context.OrgType), "op") {
		return cliConfig{}, errors.New("op commands require CLINIC_ORG_TYPE=op")
	}
	if strings.TrimSpace(cfg.Context.OrgID) == "" {
		return cliConfig{}, errors.New("CLINIC_ORG_ID is required for op commands")
	}
	cfg.Context.OrgType = "op"
	return cfg, nil
}

func resolveRequestContext(
	ctx context.Context,
	cfg cliConfig,
	cloudClusterResolver func(context.Context, clinicapi.CloudClusterLookupRequest) (clinicapi.CloudCluster, error),
) (clinicapi.RequestContext, error) {
	if strings.TrimSpace(cfg.Context.OrgID) != "" {
		return cfg.Context, nil
	}
	cluster, err := resolveCloudCluster(ctx, cfg, cloudClusterResolver)
	if err != nil {
		return clinicapi.RequestContext{}, err
	}
	return cluster.RequestContext(), nil
}

func resolveCloudCluster(
	ctx context.Context,
	cfg cliConfig,
	cloudClusterResolver func(context.Context, clinicapi.CloudClusterLookupRequest) (clinicapi.CloudCluster, error),
) (clinicapi.CloudCluster, error) {
	if strings.TrimSpace(cfg.Context.OrgID) != "" {
		return clinicapi.CloudCluster{
			ClusterID: cfg.Context.ClusterID,
			OrgID:     cfg.Context.OrgID,
		}, nil
	}
	if !strings.EqualFold(strings.TrimSpace(cfg.Context.OrgType), "cloud") {
		return clinicapi.CloudCluster{}, errors.New("org id is required unless the CLI target is cloud")
	}
	if cloudClusterResolver == nil {
		return clinicapi.CloudCluster{}, errors.New("cloud cluster resolver is required when CLINIC_ORG_ID is omitted")
	}
	cluster, err := cloudClusterResolver(ctx, clinicapi.CloudClusterLookupRequest{
		ClusterID: cfg.Context.ClusterID,
	})
	if err != nil {
		return clinicapi.CloudCluster{}, err
	}
	return cluster, nil
}

func resolveCloudTarget(
	ctx context.Context,
	cfg cliConfig,
	cloudClusterResolver func(context.Context, clinicapi.CloudClusterLookupRequest) (clinicapi.CloudCluster, error),
) (clinicapi.CloudTarget, error) {
	if strings.TrimSpace(cfg.Context.OrgID) != "" {
		return clinicapi.CloudTarget{
			OrgID:     cfg.Context.OrgID,
			ClusterID: cfg.Context.ClusterID,
		}, nil
	}
	cluster, err := resolveCloudCluster(ctx, cfg, cloudClusterResolver)
	if err != nil {
		return clinicapi.CloudTarget{}, err
	}
	return cluster.CloudTarget(), nil
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

func withDefaultEnvValue(lookup func(string) (string, bool), key, fallback string) func(string) (string, bool) {
	return func(requested string) (string, bool) {
		if value, ok := lookup(requested); ok {
			return value, ok
		}
		if requested == key {
			return fallback, true
		}
		return "", false
	}
}

func parseInt64Env(key, value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", key)
	}
	return parsed, nil
}
