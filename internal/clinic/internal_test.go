package clinic

import (
	"testing"
	"time"

	clinicapi "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

func TestSelectCatalogItemForSlowQueriesPrefersSlowLogCollector(t *testing.T) {
	item, err := selectCatalogItem(catalogIntentSlowQueries, []clinicapi.ClinicDataItem{
		{
			ItemID:     "plain-log",
			HaveLog:    true,
			Collectors: []string{"log.std"},
			StartTime:  1772776800,
			EndTime:    1772777400,
		},
		{
			ItemID:     "slow-log",
			HaveLog:    true,
			Collectors: []string{"log.std", "log.slow"},
			StartTime:  1772776800,
			EndTime:    1772777400,
		},
	}, 1772776800, 1772777400)
	if err != nil {
		t.Fatalf("selectCatalogItem failed: %v", err)
	}
	if item.ItemID != "slow-log" {
		t.Fatalf("expected slow-log, got %+v", item)
	}
}

func TestSelectCatalogItemForConfigsPrefersLatestSnapshot(t *testing.T) {
	item, err := selectCatalogItem(catalogIntentConfigs, []clinicapi.ClinicDataItem{
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
	}, 1772776800, 1772777400)
	if err != nil {
		t.Fatalf("selectCatalogItem failed: %v", err)
	}
	if item.ItemID != "latest-config" {
		t.Fatalf("expected latest-config, got %+v", item)
	}
}

func TestResolvedPlatformFromSharedMetadata(t *testing.T) {
	if got := resolvedPlatformFromSharedMetadata(
		clinicapi.CloudCluster{ClusterType: "tidb-cluster", DeployType: "tiup-cluster", DeployTypeV2: "tiup-cluster"},
		clinicapi.Org{Type: ""},
	); got != TargetPlatformTiUPCluster {
		t.Fatalf("expected tiup-cluster platform, got %s", got)
	}
	if got := resolvedPlatformFromSharedMetadata(
		clinicapi.CloudCluster{ClusterType: "cloud", DeployType: "dedicated", DeployTypeV2: "dedicated"},
		clinicapi.Org{Type: "cloud"},
	); got != TargetPlatformCloud {
		t.Fatalf("expected cloud platform, got %s", got)
	}
}

func TestBuildResolvedClusterTargetUsesPlatformSpecificOrgType(t *testing.T) {
	target := buildResolvedClusterTarget(ClusterSelector{
		Platform:  TargetPlatformTiUPCluster,
		OrgID:     "org-tiup",
		ClusterID: "cluster-9",
	}, clinicapi.CloudCluster{
		ClusterType:  "tidb-cluster",
		OrgID:        "org-tiup",
		ClusterID:    "cluster-9",
		DeployType:   "tiup-cluster",
		DeployTypeV2: "tiup-cluster",
	})
	if target.Metrics.Context.OrgType != "tidb-cluster" {
		t.Fatalf("expected collected-data org type, got %+v", target.Metrics.Context)
	}
	if target.Metrics.Context.RoutingOrgType != "op" {
		t.Fatalf("expected collected-data routing org type, got %+v", target.Metrics.Context)
	}
	if target.ClusterType != "tidb-cluster" {
		t.Fatalf("expected cluster type to be preserved, got %+v", target)
	}
}

func TestFeatureGateCapabilityOverrides(t *testing.T) {
	overrides := featureGateCapabilityOverrides(clinicapi.CloudClusterFeatureGates{
		Known:               true,
		LogsEnabled:         false,
		SlowQueryEnabled:    true,
		TopSQLEnabled:       false,
		ContinuousProfiling: false,
	})
	if got, ok := overrides[CapabilityLogs]; !ok || got.Available {
		t.Fatalf("expected logs override to disable logs, got %+v", overrides)
	}
	if _, ok := overrides[CapabilitySlowQuery]; ok {
		t.Fatalf("slow query should not be disabled when feature gate is enabled: %+v", overrides)
	}
	if got, ok := overrides[CapabilityTopSQL]; !ok || got.Reason == "" {
		t.Fatalf("expected topsql override reason, got %+v", overrides)
	}
}

func TestCatalogCapabilityOverrides(t *testing.T) {
	overrides := catalogCapabilityOverrides([]clinicapi.ClinicDataItem{
		{
			ItemID:     "diag-1",
			HaveLog:    true,
			Collectors: []string{"system", "log.slow"},
		},
		{
			ItemID:     "diag-2",
			HaveConfig: true,
		},
	})
	if got := overrides[CapabilityLogs]; !got.Available {
		t.Fatalf("expected logs to be available, got %+v", got)
	}
	if got := overrides[CapabilitySlowQuery]; !got.Available {
		t.Fatalf("expected slow query to be available, got %+v", got)
	}
	if got := overrides[CapabilityConfigs]; !got.Available {
		t.Fatalf("expected configs to be available, got %+v", got)
	}
}

func TestStaticCapabilityDescriptorsForTiUPCluster(t *testing.T) {
	descriptors := staticCapabilityDescriptors(resolvedTarget{
		Platform:     TargetPlatformTiUPCluster,
		ClusterID:    "cluster-1",
		DeployType:   "tiup-cluster",
		DeployTypeV2: "tiup-cluster",
	}, map[CapabilityName]bool{
		CapabilityClusterDetail:   true,
		CapabilityTopology:        true,
		CapabilityEvents:          true,
		CapabilityMetrics:         true,
		CapabilityLogs:            true,
		CapabilitySQLQuery:        true,
		CapabilitySchema:          true,
		CapabilityTopSQL:          true,
		CapabilitySlowQuery:       true,
		CapabilitySQLStatements:   true,
		CapabilityConfigs:         true,
		CapabilityProfiling:       true,
		CapabilityDiagnosticFiles: true,
	})
	metrics, ok := ClusterCapabilities{Capabilities: descriptors}.Lookup(CapabilityMetrics)
	if !ok || !metrics.Available {
		t.Fatalf("expected metrics to stay available for tiup-cluster, got %+v", metrics)
	}
	logs, ok := ClusterCapabilities{Capabilities: descriptors}.Lookup(CapabilityLogs)
	if !ok || logs.Available {
		t.Fatalf("expected logs to require collected data by default, got %+v", logs)
	}
	slowQuery, ok := ClusterCapabilities{Capabilities: descriptors}.Lookup(CapabilitySlowQuery)
	if !ok || slowQuery.Available {
		t.Fatalf("expected slow query to require collected data by default, got %+v", slowQuery)
	}
}

func TestParseCollectedSlowQueryTimestampAcceptsSecondsMillisecondsAndRFC3339(t *testing.T) {
	if got, err := parseCollectedSlowQueryTimestamp("timestamp", "1767090292"); err != nil || got != 1767090292 {
		t.Fatalf("expected unix seconds to parse, got value=%d err=%v", got, err)
	}
	if got, err := parseCollectedSlowQueryTimestamp("timestamp", "1767090292000"); err != nil || got != 1767090292 {
		t.Fatalf("expected unix milliseconds to parse, got value=%d err=%v", got, err)
	}
	expected := time.Date(2026, time.April, 16, 10, 24, 52, 0, time.UTC).Unix()
	if got, err := parseCollectedSlowQueryTimestamp("timestamp", "2026-04-16T10:24:52Z"); err != nil || got != expected {
		t.Fatalf("expected RFC3339 to parse, got value=%d err=%v", got, err)
	}
}

func TestMatchesCollectedSlowQuerySampleUsesDigestConnectionAndTimestamp(t *testing.T) {
	sample := clinicapi.CloudSlowQueryListEntry{
		ID:           "sq-1",
		Digest:       "digest-1",
		ConnectionID: "6982771848633227000",
		Timestamp:    "1767090292",
	}
	if !matchesCollectedSlowQuerySample(sample, SlowQueryDetailQuery{
		Digest:       "digest-1",
		ConnectionID: "6982771848633227000",
		Timestamp:    "1767090292",
	}, 1767090292) {
		t.Fatalf("expected sample to match detail query")
	}
	if matchesCollectedSlowQuerySample(sample, SlowQueryDetailQuery{
		Digest:       "digest-2",
		ConnectionID: "6982771848633227000",
		Timestamp:    "1767090292",
	}, 1767090292) {
		t.Fatalf("expected digest mismatch to fail")
	}
}
