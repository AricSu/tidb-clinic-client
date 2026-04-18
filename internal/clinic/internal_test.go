package clinic

import (
	"testing"

	clinicapi "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

func TestSelectCatalogItemForCollectedDataPrefersLargestOverlap(t *testing.T) {
	item, err := selectCatalogItem(catalogIntentCollectedData, []clinicapi.ClinicDataItem{
		{
			ItemID:    "older",
			StartTime: 1772776200,
			EndTime:   1772776500,
		},
		{
			ItemID:    "best-match",
			StartTime: 1772776800,
			EndTime:   1772777400,
		},
	}, 1772776810, 1772777390)
	if err != nil {
		t.Fatalf("selectCatalogItem failed: %v", err)
	}
	if item.ItemID != "best-match" {
		t.Fatalf("expected best-match, got %+v", item)
	}
}

func TestSelectCatalogItemForCollectedDataDefaultsToLatestWhenRangeMissing(t *testing.T) {
	item, err := selectCatalogItem(catalogIntentCollectedData, []clinicapi.ClinicDataItem{
		{
			ItemID:    "older",
			StartTime: 1772776200,
			EndTime:   1772776500,
		},
		{
			ItemID:    "latest",
			StartTime: 1772776800,
			EndTime:   1772777400,
		},
	}, 0, 0)
	if err != nil {
		t.Fatalf("selectCatalogItem failed: %v", err)
	}
	if item.ItemID != "latest" {
		t.Fatalf("expected latest, got %+v", item)
	}
}

func TestCatalogDataTypeForIntent(t *testing.T) {
	if got := catalogDataTypeForIntent(catalogIntentLogs); got != clinicapi.CatalogDataTypeLogs {
		t.Fatalf("expected logs intent to use data_type=2, got %d", got)
	}
	if got := catalogDataTypeForIntent(catalogIntentSlowQueries); got != clinicapi.CatalogDataTypeLogs {
		t.Fatalf("expected slow query intent to use data_type=2, got %d", got)
	}
	if got := catalogDataTypeForIntent(catalogIntentCollectedData); got != clinicapi.CatalogDataTypeCollectedDownload {
		t.Fatalf("expected collected-data intent to use data_type=4, got %d", got)
	}
}

func TestSelectCatalogItemForSlowQueriesPrefersLogSlowCollector(t *testing.T) {
	item, err := selectCatalogItem(catalogIntentSlowQueries, []clinicapi.ClinicDataItem{
		{
			ItemID:     "general-log-item",
			Collectors: []string{"log.general"},
			HaveLog:    true,
			StartTime:  1772776800,
			EndTime:    1772777400,
		},
		{
			ItemID:     "slow-log-item",
			Collectors: []string{"log.slow"},
			HaveLog:    true,
			StartTime:  1772776800,
			EndTime:    1772777400,
		},
	}, 1772776810, 1772777390)
	if err != nil {
		t.Fatalf("selectCatalogItem failed: %v", err)
	}
	if item.ItemID != "slow-log-item" {
		t.Fatalf("expected slow-log-item, got %+v", item)
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
