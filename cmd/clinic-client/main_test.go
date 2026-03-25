package main

import (
	"context"
	"strings"
	"testing"
	"time"

	clinicapi "github.com/aric/tidb-clinic-client"
)

func TestLoadConfigFromEnvRequiresAPIKeyAndClusterID(t *testing.T) {
	_, err := loadConfigFromEnv(func(key string) (string, bool) {
		return "", false
	}, func() time.Time {
		return time.Unix(1772777400, 0)
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "CLINIC_API_KEY") {
		t.Fatalf("expected missing api key error, got=%v", err)
	}
}

func TestLoadConfigFromEnvAppliesDefaults(t *testing.T) {
	lookup := func(key string) (string, bool) {
		switch key {
		case "CLINIC_API_KEY":
			return "token", true
		case "CLINIC_CLUSTER_ID":
			return "cluster-9", true
		default:
			return "", false
		}
	}
	cfg, err := loadConfigFromEnv(lookup, func() time.Time {
		return time.Unix(1772777400, 0)
	})
	if err != nil {
		t.Fatalf("loadConfigFromEnv failed: %v", err)
	}
	if cfg.BaseURL != "https://clinic.pingcap.com" {
		t.Fatalf("expected default base url, got=%q", cfg.BaseURL)
	}
	if cfg.Context.OrgType != "cloud" {
		t.Fatalf("expected default org type cloud, got=%q", cfg.Context.OrgType)
	}
	if cfg.Context.OrgID != "" {
		t.Fatalf("expected org id to remain unset until resolved, got=%q", cfg.Context.OrgID)
	}
	if cfg.Query != "sum(tidb_server_connections)" {
		t.Fatalf("unexpected default query: %q", cfg.Query)
	}
	if cfg.Step != "1m" {
		t.Fatalf("unexpected default step: %q", cfg.Step)
	}
	if cfg.Start != 1772776800 {
		t.Fatalf("unexpected default start: %d", cfg.Start)
	}
	if cfg.End != 1772777400 {
		t.Fatalf("unexpected default end: %d", cfg.End)
	}
	if cfg.Timeout != 20*time.Second {
		t.Fatalf("unexpected default timeout: %v", cfg.Timeout)
	}
}

func TestLoadConfigFromEnvRespectsOptionalOverrides(t *testing.T) {
	lookup := func(key string) (string, bool) {
		switch key {
		case "CLINIC_API_BASE_URL":
			return "https://clinic.pingcap.com", true
		case "CLINIC_API_KEY":
			return "token", true
		case "CLINIC_ORG_TYPE":
			return "op", true
		case "CLINIC_ORG_ID":
			return "org-2", true
		case "CLINIC_CLUSTER_ID":
			return "cluster-2", true
		case "CLINIC_METRICS_QUERY":
			return "sum(tidb_server_maxprocs)", true
		case "CLINIC_RANGE_START":
			return "1772776500", true
		case "CLINIC_RANGE_END":
			return "1772777100", true
		case "CLINIC_RANGE_STEP":
			return "30s", true
		case "CLINIC_TIMEOUT_SEC":
			return "7", true
		default:
			return "", false
		}
	}
	cfg, err := loadConfigFromEnv(lookup, func() time.Time {
		return time.Unix(1772777400, 0)
	})
	if err != nil {
		t.Fatalf("loadConfigFromEnv failed: %v", err)
	}
	if cfg.Query != "sum(tidb_server_maxprocs)" {
		t.Fatalf("unexpected query override: %q", cfg.Query)
	}
	if cfg.Step != "30s" {
		t.Fatalf("unexpected step override: %q", cfg.Step)
	}
	if cfg.Start != 1772776500 || cfg.End != 1772777100 {
		t.Fatalf("unexpected range override: start=%d end=%d", cfg.Start, cfg.End)
	}
	if cfg.Timeout != 7*time.Second {
		t.Fatalf("unexpected timeout override: %v", cfg.Timeout)
	}
}

func TestResolveRequestContextUsesCloudResolverWhenOrgIDMissing(t *testing.T) {
	cfg := cliConfig{
		Context: clinicapi.RequestContext{
			OrgType:   "cloud",
			ClusterID: "cluster-9",
		},
	}
	var gotRequest clinicapi.CloudClusterLookupRequest
	resolved, err := resolveRequestContext(context.Background(), cfg, func(ctx context.Context, req clinicapi.CloudClusterLookupRequest) (clinicapi.CloudCluster, error) {
		gotRequest = req
		return clinicapi.CloudCluster{
			ClusterID:  "cluster-9",
			OrgID:      "org-1",
			TenantID:   "tenant-1",
			ProjectID:  "project-1",
			Provider:   "aws",
			Region:     "us-east-1",
			DeployType: "dedicated",
		}, nil
	})
	if err != nil {
		t.Fatalf("resolveRequestContext failed: %v", err)
	}
	if gotRequest.ClusterID != "cluster-9" {
		t.Fatalf("unexpected lookup request: %+v", gotRequest)
	}
	if resolved.OrgType != "cloud" || resolved.OrgID != "org-1" || resolved.ClusterID != "cluster-9" {
		t.Fatalf("unexpected resolved request context: %+v", resolved)
	}
}

func TestResolveRequestContextReturnsExistingContextWithoutResolver(t *testing.T) {
	cfg := cliConfig{
		Context: clinicapi.RequestContext{
			OrgType:   "op",
			OrgID:     "org-2",
			ClusterID: "cluster-2",
		},
	}
	resolved, err := resolveRequestContext(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("resolveRequestContext failed: %v", err)
	}
	if resolved != cfg.Context {
		t.Fatalf("expected unchanged context, got=%+v want=%+v", resolved, cfg.Context)
	}
}

func TestSelectCatalogItemForOPLogsPrefersBestOverlappingLogItem(t *testing.T) {
	items := []clinicapi.ClinicDataItem{
		{
			ItemID:     "older-log",
			Collectors: []string{"log.std"},
			HaveLog:    true,
			StartTime:  1772776200,
			EndTime:    1772776500,
		},
		{
			ItemID:     "best-log",
			Collectors: []string{"log.std"},
			HaveLog:    true,
			StartTime:  1772776800,
			EndTime:    1772777400,
		},
	}

	got, err := selectCatalogItemForOP(opCatalogIntentLogs, items, 1772776800, 1772777400)
	if err != nil {
		t.Fatalf("selectCatalogItemForOP failed: %v", err)
	}
	if got.ItemID != "best-log" {
		t.Fatalf("expected best-log, got=%+v", got)
	}
}

func TestSelectCatalogItemForOPSlowQueriesPrefersLogSlowCollector(t *testing.T) {
	items := []clinicapi.ClinicDataItem{
		{
			ItemID:     "plain-log",
			Collectors: []string{"log.std"},
			HaveLog:    true,
			StartTime:  1772776800,
			EndTime:    1772777400,
		},
		{
			ItemID:     "slow-log",
			Collectors: []string{"log.std", "log.slow"},
			HaveLog:    true,
			StartTime:  1772776800,
			EndTime:    1772777400,
		},
	}

	got, err := selectCatalogItemForOP(opCatalogIntentSlowQueries, items, 1772776800, 1772777400)
	if err != nil {
		t.Fatalf("selectCatalogItemForOP failed: %v", err)
	}
	if got.ItemID != "slow-log" {
		t.Fatalf("expected slow-log, got=%+v", got)
	}
}

func TestSelectCatalogItemForOPConfigsPrefersLatestConfigSnapshot(t *testing.T) {
	items := []clinicapi.ClinicDataItem{
		{
			ItemID:     "older-config",
			HaveConfig: true,
			StartTime:  1772776200,
			EndTime:    1772776500,
		},
		{
			ItemID:     "latest-config",
			HaveConfig: true,
			StartTime:  1772776800,
			EndTime:    1772777400,
		},
	}

	got, err := selectCatalogItemForOP(opCatalogIntentConfigs, items, 1772776800, 1772777400)
	if err != nil {
		t.Fatalf("selectCatalogItemForOP failed: %v", err)
	}
	if got.ItemID != "latest-config" {
		t.Fatalf("expected latest-config, got=%+v", got)
	}
}

func TestSelectCatalogItemForOPReturnsErrorWhenNoEligibleItem(t *testing.T) {
	items := []clinicapi.ClinicDataItem{
		{
			ItemID:     "config-only",
			HaveConfig: true,
			StartTime:  1772776800,
			EndTime:    1772777400,
		},
	}

	_, err := selectCatalogItemForOP(opCatalogIntentLogs, items, 1772776800, 1772777400)
	if err == nil {
		t.Fatalf("expected no eligible item error")
	}
	if !strings.Contains(err.Error(), "no suitable catalog item") {
		t.Fatalf("unexpected error: %v", err)
	}
}
