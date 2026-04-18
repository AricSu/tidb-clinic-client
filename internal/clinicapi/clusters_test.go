package clinicapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestDecodeCloudClusterLookupItemTreatsZeroDeletedAtAsActive(t *testing.T) {
	cluster := decodeClusterLookupItem(clusterLookupItem{
		ClusterID:   "cluster-1",
		OrgID:       "org-1",
		ClusterType: "tidb-cluster",
		DeletedAt:   0,
		Status:      "active",
	})
	if cluster.DeletedAt != nil {
		t.Fatalf("expected DeletedAt to be nil when clusterDeletedAt is 0, got %v", *cluster.DeletedAt)
	}
	if cluster.ClusterType != "tidb-cluster" {
		t.Fatalf("expected cluster type to be preserved, got %+v", cluster)
	}
}

func TestGetClusterUsesDashboardClustersAndFiltersExactMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/clinic/api/v1/dashboard/clusters" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		assertClusterLookupQuery(t, r.URL.Query(), "cluster-1", true, "10", "1")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"clusterID":"cluster-10","clusterName":"fuzzy","orgID":"org-10","clusterType":"tidb-cluster"},{"clusterID":"cluster-1","clusterName":"exact","orgID":"org-1","clusterType":"tidb-cluster"}]}`))
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "test-token",
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	cluster, err := client.GetCluster(context.Background(), CloudClusterLookupRequest{
		ClusterID:   "cluster-1",
		ShowDeleted: true,
	})
	if err != nil {
		t.Fatalf("GetCluster failed: %v", err)
	}
	if cluster.ClusterID != "cluster-1" || cluster.OrgID != "org-1" {
		t.Fatalf("unexpected cluster: %+v", cluster)
	}
}

func TestGetClusterReturnsNotFoundWithoutExactMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"clusterID":"cluster-10","clusterName":"fuzzy","orgID":"org-10","clusterType":"tidb-cluster"}]}`))
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "test-token",
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.GetCluster(context.Background(), CloudClusterLookupRequest{
		ClusterID:   "cluster-1",
		ShowDeleted: true,
	})
	if err == nil {
		t.Fatalf("expected not found error")
	}
	clinicErr, ok := err.(*Error)
	if !ok || clinicErr.Class != ErrNotFound {
		t.Fatalf("expected not found clinic error, got %T %v", err, err)
	}
}

func TestGetClusterReturnsInvalidRequestForMultipleExactMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"clusterID":"cluster-1","clusterName":"exact-a","orgID":"org-a","clusterType":"tidb-cluster"},{"clusterID":"cluster-1","clusterName":"exact-b","orgID":"org-b","clusterType":"tidb-cluster"}]}`))
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "test-token",
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.GetCluster(context.Background(), CloudClusterLookupRequest{
		ClusterID:   "cluster-1",
		ShowDeleted: true,
	})
	if err == nil {
		t.Fatalf("expected invalid request error")
	}
	clinicErr, ok := err.(*Error)
	if !ok || clinicErr.Class != ErrInvalidRequest {
		t.Fatalf("expected invalid request clinic error, got %T %v", err, err)
	}
}

func assertClusterLookupQuery(t *testing.T, values url.Values, expectedQuery string, showDeleted bool, limit, page string) {
	t.Helper()
	if got := values.Get("query"); got != expectedQuery {
		t.Fatalf("unexpected query value: %q", got)
	}
	if got := values.Get("show_deleted"); got != strings.ToLower(strconv.FormatBool(showDeleted)) {
		t.Fatalf("unexpected show_deleted value: %q", got)
	}
	if got := values.Get("sort"); got != "" {
		t.Fatalf("unexpected sort value: %q", got)
	}
	if got := values.Get("order"); got != "" {
		t.Fatalf("unexpected order value: %q", got)
	}
	if got := values.Get("limit"); got != limit {
		t.Fatalf("unexpected limit value: %q", got)
	}
	if got := values.Get("page"); got != page {
		t.Fatalf("unexpected page value: %q", got)
	}
	if got := values.Get("cluster_id"); got != "" {
		t.Fatalf("cluster_id should not be sent, got %q", got)
	}
}

func TestDecodeCloudClusterDetailFeatureGates(t *testing.T) {
	detail := decodeCloudClusterDetail(map[string]any{
		"id":   "cluster-1",
		"name": "demo",
		"featureGates": map[string]any{
			"logsEnabled":    false,
			"conProfEnabled": true,
		},
	})
	if !detail.FeatureGates.Known {
		t.Fatalf("expected feature gates to be marked known")
	}
	if detail.FeatureGates.LogsEnabled {
		t.Fatalf("expected logs gate to be false, got %+v", detail.FeatureGates)
	}
	if !detail.FeatureGates.ContinuousProfiling {
		t.Fatalf("expected continuous profiling gate to be true, got %+v", detail.FeatureGates)
	}
}
