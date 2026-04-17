package cli

import (
	"strings"
	"testing"
	"time"
)

func TestLoadCloudSlowQueryDetailConfigSupportsIDOnly(t *testing.T) {
	cfg, err := loadCloudSlowQueryDetailConfig(testSlowQueryLookup(
		"CLINIC_SLOWQUERY_ID", "slowquery-1",
		"CLINIC_SLOWQUERY_DIGEST", "stale-digest",
		"CLINIC_SLOWQUERY_CONNECTION_ID", "stale-connection",
		"CLINIC_SLOWQUERY_TIMESTAMP", "1767089088",
	), testNow)
	if err != nil {
		t.Fatalf("loadCloudSlowQueryDetailConfig failed: %v", err)
	}
	if cfg.ID != "slowquery-1" {
		t.Fatalf("unexpected id: %q", cfg.ID)
	}
	if cfg.Digest != "" || cfg.ConnectionID != "" || cfg.Timestamp != "" {
		t.Fatalf("unexpected compatibility fields: %+v", cfg)
	}
}

func TestLoadCloudSlowQueryDetailConfigSupportsSemanticTriple(t *testing.T) {
	cfg, err := loadCloudSlowQueryDetailConfig(testSlowQueryLookup(
		"CLINIC_SLOWQUERY_DIGEST", "digest-1",
		"CLINIC_SLOWQUERY_CONNECTION_ID", "conn-1",
		"CLINIC_SLOWQUERY_TIMESTAMP", "1767089088",
	), testNow)
	if err != nil {
		t.Fatalf("loadCloudSlowQueryDetailConfig failed: %v", err)
	}
	if cfg.Digest != "digest-1" || cfg.ConnectionID != "conn-1" || cfg.Timestamp != "1767089088" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadCloudSlowQueryDetailConfigRejectsMissingDetailLocator(t *testing.T) {
	_, err := loadCloudSlowQueryDetailConfig(testSlowQueryLookup(), testNow)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "CLINIC_SLOWQUERY_ID") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadCloudSlowQueryListConfigUsesSharedRangeInputs(t *testing.T) {
	cfg, err := loadCloudSlowQueryListConfig(testSlowQueryLookup(
		"CLINIC_SLOWQUERY_DIGEST", "digest-2",
		"CLINIC_SLOWQUERY_ORDER_BY", "query_time",
		"CLINIC_LIMIT", "20",
		"CLINIC_DESC", "true",
	), testNow)
	if err != nil {
		t.Fatalf("loadCloudSlowQueryListConfig failed: %v", err)
	}
	if cfg.Start != "1772776800" || cfg.End != "1772777400" {
		t.Fatalf("unexpected range: start=%s end=%s", cfg.Start, cfg.End)
	}
	if cfg.Digest != "digest-2" || cfg.OrderBy != "query_time" || cfg.Limit != 20 || !cfg.Desc {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func testSlowQueryLookup(values ...string) func(string) (string, bool) {
	entries := map[string]string{
		"CLINIC_API_KEY":     "token",
		"CLINIC_CLUSTER_ID":  "cluster-9",
		"CLINIC_RANGE_START": "1772776800",
		"CLINIC_RANGE_END":   "1772777400",
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
