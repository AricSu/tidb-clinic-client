package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	if len(cfg.Match) != 0 {
		t.Fatalf("expected no matchers by default, got=%v", cfg.Match)
	}
	if cfg.Time != 1772777100 {
		t.Fatalf("unexpected default query time: %d", cfg.Time)
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

func TestLoadConfigFromEnvParsesMetricMatchersAndQueryTime(t *testing.T) {
	lookup := func(key string) (string, bool) {
		switch key {
		case "CLINIC_API_KEY":
			return "token", true
		case "CLINIC_CLUSTER_ID":
			return "cluster-9", true
		case "CLINIC_METRICS_MATCH":
			return "metric_a{instance=\"tidb-0\"}\nmetric_a{instance=\"tidb-1\"}", true
		case "CLINIC_QUERY_TIME":
			return "1772777000", true
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
	if len(cfg.Match) != 2 || cfg.Match[0] != `metric_a{instance="tidb-0"}` || cfg.Match[1] != `metric_a{instance="tidb-1"}` {
		t.Fatalf("unexpected matchers: %+v", cfg.Match)
	}
	if cfg.Time != 1772777000 {
		t.Fatalf("unexpected query time: %d", cfg.Time)
	}
}

func TestLookupEnvWithDotEnvFallsBackToDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := strings.Join([]string{
		"CLINIC_API_KEY=token-from-file",
		"CLINIC_CLUSTER_ID=cluster-9",
		`CLINIC_METRICS_QUERY="sum(rate(foo_total{type='$command'}[1m]))"`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	lookup := lookupEnvWithDotEnv(func(string) (string, bool) {
		return "", false
	}, path)

	if value, ok := lookup("CLINIC_API_KEY"); !ok || value != "token-from-file" {
		t.Fatalf("unexpected api key: ok=%v value=%q", ok, value)
	}
	if value, ok := lookup("CLINIC_METRICS_QUERY"); !ok || value != "sum(rate(foo_total{type='$command'}[1m]))" {
		t.Fatalf("unexpected metrics query: ok=%v value=%q", ok, value)
	}
}

func TestLookupEnvWithDotEnvPrefersRealEnvironment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("CLINIC_API_KEY=token-from-file\n"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	lookup := lookupEnvWithDotEnv(func(key string) (string, bool) {
		if key == "CLINIC_API_KEY" {
			return "token-from-env", true
		}
		return "", false
	}, path)

	if value, ok := lookup("CLINIC_API_KEY"); !ok || value != "token-from-env" {
		t.Fatalf("unexpected api key precedence: ok=%v value=%q", ok, value)
	}
}
