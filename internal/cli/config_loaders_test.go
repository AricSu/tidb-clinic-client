package cli

import (
	"strings"
	"testing"
	"time"
)

func TestLoadCloudEventListConfigUsesFilters(t *testing.T) {
	cfg, err := loadCloudEventListConfig(testSlowQueryLookup(
		"CLINIC_EVENT_NAME", "backup",
		"CLINIC_EVENT_SEVERITY", "warning",
	), testNow)
	if err != nil {
		t.Fatalf("loadCloudEventListConfig failed: %v", err)
	}
	if cfg.Name != "backup" {
		t.Fatalf("unexpected event name: %q", cfg.Name)
	}
	if cfg.Severity == nil || *cfg.Severity != 1 {
		t.Fatalf("unexpected event severity: %v", cfg.Severity)
	}
}

func TestLoadCloudEventListConfigRejectsUnknownSeverity(t *testing.T) {
	_, err := loadCloudEventListConfig(testSlowQueryLookup(
		"CLINIC_EVENT_SEVERITY", "fatal",
	), testNow)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "CLINIC_EVENT_SEVERITY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadCloudEventListConfigUsesActiveFlagInputs(t *testing.T) {
	restore := pushEventFlagInputs(eventFlagInputs{
		Name:        "%backup%",
		NameSet:     true,
		Severity:    "critical",
		SeveritySet: true,
	})
	defer restore()

	cfg, err := loadCloudEventListConfig(testSlowQueryLookup(
		"CLINIC_EVENT_NAME", "from-env",
		"CLINIC_EVENT_SEVERITY", "warning",
	), testNow)
	if err != nil {
		t.Fatalf("loadCloudEventListConfig failed: %v", err)
	}
	if cfg.Name != "%backup%" {
		t.Fatalf("unexpected event name: %q", cfg.Name)
	}
	if cfg.Severity == nil || *cfg.Severity != 3 {
		t.Fatalf("unexpected event severity: %v", cfg.Severity)
	}
}

func TestLoadCloudLogsConfigUsesQueryFlagInputs(t *testing.T) {
	restore := pushCloudLogsFlagInputs(cloudLogsFlagInputs{
		Query:        `{container="tidb"} |= "ERROR"`,
		QuerySet:     true,
		Limit:        25,
		LimitSet:     true,
		Direction:    "forward",
		DirectionSet: true,
	})
	defer restore()

	cfg, err := loadCloudLogsConfig(testSlowQueryLookup(
		"CLINIC_LOKI_QUERY", "from-env",
		"CLINIC_LOKI_LIMIT", "10",
		"CLINIC_LOKI_DIRECTION", "backward",
	), testNow)
	if err != nil {
		t.Fatalf("loadCloudLogsConfig failed: %v", err)
	}
	if cfg.Mode != cloudLogsModeQueryRange {
		t.Fatalf("unexpected mode: %q", cfg.Mode)
	}
	if cfg.Query != `{container="tidb"} |= "ERROR"` || cfg.Limit != 25 || cfg.Direction != "forward" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadCloudLogsConfigUsesLabelFlagInputs(t *testing.T) {
	restore := pushCloudLogsFlagInputs(cloudLogsFlagInputs{
		LabelName:    "container",
		LabelNameSet: true,
	})
	defer restore()

	cfg, err := loadCloudLogsConfig(testSlowQueryLookup(
		"CLINIC_LOKI_LABEL", "instance",
	), testNow)
	if err != nil {
		t.Fatalf("loadCloudLogsConfig failed: %v", err)
	}
	if cfg.Mode != cloudLogsModeLabelValues {
		t.Fatalf("unexpected mode: %q", cfg.Mode)
	}
	if cfg.LabelName != "container" {
		t.Fatalf("unexpected label name: %q", cfg.LabelName)
	}
}

func TestLoadCloudLogsConfigDefaultsToLabels(t *testing.T) {
	cfg, err := loadCloudLogsConfig(testSlowQueryLookup(), testNow)
	if err != nil {
		t.Fatalf("loadCloudLogsConfig failed: %v", err)
	}
	if cfg.Mode != cloudLogsModeLabels {
		t.Fatalf("unexpected mode: %q", cfg.Mode)
	}
}

func TestLoadCloudLogsConfigRejectsMixedInputs(t *testing.T) {
	restore := pushCloudLogsFlagInputs(cloudLogsFlagInputs{
		Query:        `{container="tidb"}`,
		QuerySet:     true,
		LabelName:    "container",
		LabelNameSet: true,
	})
	defer restore()

	_, err := loadCloudLogsConfig(testSlowQueryLookup(), testNow)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "cannot combine") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadMetricsCompileConfigUsesFlagInputs(t *testing.T) {
	restore := pushMetricsCompileFlagInputs(metricsCompileFlagInputs{
		ExprDescription:    "P99 query latency for each TiDB instance",
		ExprDescriptionSet: true,
	})
	defer restore()

	cfg, err := loadMetricsCompileConfig(testSlowQueryLookup(
		"CLINIC_METRICS_EXPR_DESCRIPTION", "from-env",
	), testNow)
	if err != nil {
		t.Fatalf("loadMetricsCompileConfig failed: %v", err)
	}
	if cfg.ExprDescription != "P99 query latency for each TiDB instance" {
		t.Fatalf("unexpected expr description: %q", cfg.ExprDescription)
	}
}

func TestLoadSlowQueryConfigUsesSpecificEnvInputs(t *testing.T) {
	cfg, err := loadSlowQueryConfig(testSlowQueryLookup(
		"CLINIC_SLOWQUERY_DIGEST", "digest-1",
		"CLINIC_SLOWQUERY_ORDER_BY", "queryTime",
		"CLINIC_SLOWQUERY_LIMIT", "20",
		"CLINIC_SLOWQUERY_DESC", "true",
		"CLINIC_SLOWQUERY_FIELDS", "query,timestamp",
	), testNow)
	if err != nil {
		t.Fatalf("loadSlowQueryConfig failed: %v", err)
	}
	if cfg.Digest != "digest-1" || cfg.OrderBy != "queryTime" || cfg.Limit != 20 || !cfg.Desc {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if got := strings.Join(cfg.Fields, ","); got != "query,timestamp" {
		t.Fatalf("unexpected fields: %+v", cfg.Fields)
	}
}

func TestLoadSlowQueryConfigUsesSearchDefaultsWhenInputsMissing(t *testing.T) {
	cfg, err := loadSlowQueryConfig(testSlowQueryLookup(), testNow)
	if err != nil {
		t.Fatalf("loadSlowQueryConfig failed: %v", err)
	}
	if cfg.OrderBy != "query_time" || cfg.Limit != 100 || cfg.Desc {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadSlowQueryConfigFallsBackToLegacyEnvInputs(t *testing.T) {
	cfg, err := loadSlowQueryConfig(testSlowQueryLookup(
		"CLINIC_LIMIT", "15",
		"CLINIC_DESC", "true",
	), testNow)
	if err != nil {
		t.Fatalf("loadSlowQueryConfig failed: %v", err)
	}
	if cfg.Limit != 15 || !cfg.Desc {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadSlowQueryConfigUsesFlagInputs(t *testing.T) {
	restore := pushSlowQueryFlagInputs(slowQueryFlagInputs{
		Digest:     "digest-2",
		DigestSet:  true,
		OrderBy:    "execCount",
		OrderBySet: true,
		Limit:      5,
		LimitSet:   true,
		Desc:       true,
		DescSet:    true,
		Fields:     "query,connection_id",
		FieldsSet:  true,
	})
	defer restore()

	cfg, err := loadSlowQueryConfig(testSlowQueryLookup(
		"CLINIC_SLOWQUERY_ORDER_BY", "queryTime",
		"CLINIC_SLOWQUERY_LIMIT", "20",
	), testNow)
	if err != nil {
		t.Fatalf("loadSlowQueryConfig failed: %v", err)
	}
	if cfg.Digest != "digest-2" || cfg.OrderBy != "execCount" || cfg.Limit != 5 || !cfg.Desc {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if got := strings.Join(cfg.Fields, ","); got != "query,connection_id" {
		t.Fatalf("unexpected fields: %+v", cfg.Fields)
	}
}

func TestLoadSlowQueryConfigRejectsInvalidLimit(t *testing.T) {
	_, err := loadSlowQueryConfig(testSlowQueryLookup(
		"CLINIC_SLOWQUERY_LIMIT", "0",
	), testNow)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "CLINIC_SLOWQUERY_LIMIT") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testSlowQueryLookup(values ...string) func(string) (string, bool) {
	entries := map[string]string{
		"CLINIC_API_KEY":    "token",
		"CLINIC_PORTAL_URL": "https://clinic.pingcap.com/portal/#/orgs/1372813089196930348/clusters/7372714695339837431?from=1772776800&to=1772777400",
	}
	for i := 0; i+1 < len(values); i += 2 {
		entries[values[i]] = values[i+1]
	}
	return func(key string) (string, bool) {
		value, ok := entries[key]
		return value, ok
	}
}

func testNow() time.Time {
	return time.Unix(1772777400, 0)
}
