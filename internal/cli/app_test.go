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
	if !strings.Contains(err.Error(), "CLINIC_PORTAL_URL") {
		t.Fatalf("expected missing portal url error, got=%v", err)
	}
}

func TestLoadConfigFromEnvAppliesDefaultsFromPortalURL(t *testing.T) {
	lookup := func(key string) (string, bool) {
		switch key {
		case "CLINIC_PORTAL_URL":
			return "https://clinic.pingcap.com/portal/#/orgs/1372813089196930499/clusters/10989049060142230334", true
		case "CLINIC_API_KEY":
			return "token", true
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
		t.Fatalf("expected portal base url, got=%q", cfg.BaseURL)
	}
	if cfg.OrgID != "1372813089196930499" {
		t.Fatalf("unexpected org id: %q", cfg.OrgID)
	}
	if cfg.ClusterID != "10989049060142230334" {
		t.Fatalf("unexpected cluster id: %q", cfg.ClusterID)
	}
	if cfg.Query != "histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le, instance))" {
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
}

func TestLoadConfigFromEnvRespectsOptionalQueryOverrides(t *testing.T) {
	lookup := func(key string) (string, bool) {
		switch key {
		case "CLINIC_PORTAL_URL":
			return "https://clinic.pingcap.com/portal/#/orgs/1372813089196930348/clusters/7372714695339837431?from=1773547800&to=1773548100", true
		case "CLINIC_API_KEY":
			return "token", true
		case "CLINIC_METRICS_QUERY":
			return "sum(tidb_server_maxprocs)", true
		case "CLINIC_METRICS_EXPR_DESCRIPTION":
			return "maxprocs across tidb", true
		case "CLINIC_RANGE_STEP":
			return "30s", true
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
	if cfg.ExprDescription != "maxprocs across tidb" {
		t.Fatalf("unexpected expr description override: %q", cfg.ExprDescription)
	}
	if cfg.Step != "30s" {
		t.Fatalf("unexpected step override: %q", cfg.Step)
	}
	if cfg.Start != 1773547800 || cfg.End != 1773548100 {
		t.Fatalf("expected range from portal url: start=%d end=%d", cfg.Start, cfg.End)
	}
}

func TestLoadConfigFromEnvUsesGlobalAPIKeyForCloudPortalURL(t *testing.T) {
	lookup := func(key string) (string, bool) {
		switch key {
		case "CLINIC_PORTAL_URL":
			return "https://clinic.pingcap.com/portal/#/orgs/1372813089196930499/clusters/10989049060142230334", true
		case "CLINIC_API_KEY":
			return "token-global", true
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
		t.Fatalf("unexpected base url: %q", cfg.BaseURL)
	}
	if cfg.ClusterID != "10989049060142230334" {
		t.Fatalf("unexpected cluster id: %q", cfg.ClusterID)
	}
	if cfg.APIKey != "token-global" {
		t.Fatalf("unexpected api key: %q", cfg.APIKey)
	}
	if cfg.Start != 1772776800 || cfg.End != 1772777400 {
		t.Fatalf("expected default range when portal URL has no time window, got %d..%d", cfg.Start, cfg.End)
	}
}

func TestLoadConfigFromEnvUsesCNAPIKeyForCNTiUPPortalURL(t *testing.T) {
	lookup := func(key string) (string, bool) {
		switch key {
		case "CLINIC_PORTAL_URL":
			return "https://clinic.pingcap.com.cn/portal/#/orgs/1075/clusters/7460723698814898616?from=1767679200&to=1767682800", true
		case "CLINIC_CN_API_KEY":
			return "token-cn", true
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
	if cfg.BaseURL != "https://clinic.pingcap.com.cn" {
		t.Fatalf("unexpected base url: %q", cfg.BaseURL)
	}
	if cfg.OrgID != "1075" {
		t.Fatalf("unexpected org id: %q", cfg.OrgID)
	}
	if cfg.ClusterID != "7460723698814898616" {
		t.Fatalf("unexpected cluster id: %q", cfg.ClusterID)
	}
	if cfg.APIKey != "token-cn" {
		t.Fatalf("expected cn api key, got %q", cfg.APIKey)
	}
	if cfg.Start != 1767679200 || cfg.End != 1767682800 {
		t.Fatalf("expected range from cn portal URL, got %d..%d", cfg.Start, cfg.End)
	}
}

func TestLoadConfigFromEnvRequiresCNAPIKeyForCNPortalURL(t *testing.T) {
	_, err := loadConfigFromEnv(func(key string) (string, bool) {
		switch key {
		case "CLINIC_PORTAL_URL":
			return "https://clinic.pingcap.com.cn/portal/#/orgs/1075/clusters/7460723698814898616?from=1767679200&to=1767682800", true
		case "CLINIC_API_KEY":
			return "token-global", true
		default:
			return "", false
		}
	}, func() time.Time {
		return time.Unix(1772777400, 0)
	})
	if err == nil {
		t.Fatalf("expected missing cn api key error")
	}
	if !strings.Contains(err.Error(), "CLINIC_CN_API_KEY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigFromEnvUsesGlobalAPIKeyForGlobalTiUPPortalURL(t *testing.T) {
	lookup := func(key string) (string, bool) {
		switch key {
		case "CLINIC_PORTAL_URL":
			return "https://clinic.pingcap.com/portal/#/orgs/1372813089196930348/clusters/7372714695339837431?from=1773547800&to=1773548100", true
		case "CLINIC_API_KEY":
			return "token-global", true
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
		t.Fatalf("unexpected base url: %q", cfg.BaseURL)
	}
	if cfg.OrgID != "1372813089196930348" {
		t.Fatalf("unexpected org id: %q", cfg.OrgID)
	}
	if cfg.ClusterID != "7372714695339837431" {
		t.Fatalf("unexpected cluster id: %q", cfg.ClusterID)
	}
	if cfg.APIKey != "token-global" {
		t.Fatalf("unexpected api key: %q", cfg.APIKey)
	}
	if cfg.Start != 1773547800 || cfg.End != 1773548100 {
		t.Fatalf("expected range from global tiup portal URL, got %d..%d", cfg.Start, cfg.End)
	}
}

func TestLoadConfigFromEnvUsesPortalURLAsSingleTargetInput(t *testing.T) {
	lookup := func(key string) (string, bool) {
		switch key {
		case "CLINIC_PORTAL_URL":
			return "https://clinic.pingcap.com/portal/#/orgs/1372813089196930348/clusters/7372714695339837431?from=1773547800&to=1773548100", true
		case "CLINIC_API_KEY":
			return "token-global", true
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
		t.Fatalf("expected portal base url, got %q", cfg.BaseURL)
	}
	if cfg.ClusterID != "7372714695339837431" {
		t.Fatalf("expected portal cluster id, got %q", cfg.ClusterID)
	}
	if cfg.Start != 1773547800 || cfg.End != 1773548100 {
		t.Fatalf("expected portal range, got %d..%d", cfg.Start, cfg.End)
	}
}

func TestLookupEnvWithDotEnvFallsBackToDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := strings.Join([]string{
		"CLINIC_API_KEY=token-from-file",
		"CLINIC_PORTAL_URL=https://clinic.pingcap.com/portal/#/orgs/1372813089196930499/clusters/10989049060142230334",
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
	if value, ok := lookup("CLINIC_PORTAL_URL"); !ok || !strings.Contains(value, "10989049060142230334") {
		t.Fatalf("unexpected portal url: ok=%v value=%q", ok, value)
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
