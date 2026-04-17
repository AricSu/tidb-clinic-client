package clinicapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListCollectedSlowQueriesUsesClusterScopedEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/clinic/api/v1/orgs/org-1/clusters/cluster-9/slowqueries" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("itemID"); got != "item-1" {
			t.Fatalf("unexpected itemID: %s", got)
		}
		if got := r.URL.Query().Get("begin_time"); got != "1767087000" {
			t.Fatalf("unexpected begin_time: %s", got)
		}
		if got := r.URL.Query().Get("end_time"); got != "1767090600" {
			t.Fatalf("unexpected end_time: %s", got)
		}
		if got := r.URL.Query().Get("digest"); got != "digest-1" {
			t.Fatalf("unexpected digest: %s", got)
		}
		if got := r.URL.Query().Get("orderBy"); got != "timestamp" {
			t.Fatalf("unexpected orderBy: %s", got)
		}
		if got := r.URL.Query().Get("desc"); got != "true" {
			t.Fatalf("unexpected desc: %s", got)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Fatalf("unexpected limit: %s", got)
		}
		if got := r.Header.Get("X-OrgType"); got != "op" {
			t.Fatalf("unexpected X-OrgType: %s", got)
		}
		if got := r.Header.Get("X-OrgID"); got != "org-1" {
			t.Fatalf("unexpected X-OrgID: %s", got)
		}
		if got := r.Header.Get("X-ClusterID"); got != "cluster-9" {
			t.Fatalf("unexpected X-ClusterID: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"sq-1","digest":"digest-1","query":"select 1","timestamp":1767090292,"query_time":5.062075558,"memory_max":25800,"request_count":25,"connection_id":6982771848633227000}]`))
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	result, err := client.ListCollectedSlowQueries(context.Background(), CollectedSlowQueryListRequest{
		Context: RequestContext{
			OrgType:        "tidb-cluster",
			RoutingOrgType: "op",
			OrgID:          "org-1",
			ClusterID:      "cluster-9",
		},
		ItemID:    "item-1",
		StartTime: 1767087000,
		EndTime:   1767090600,
		Digest:    "digest-1",
		OrderBy:   "timestamp",
		Desc:      true,
		Limit:     100,
	})
	if err != nil {
		t.Fatalf("ListCollectedSlowQueries failed: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected one row, got %d", len(result))
	}
	if result[0].ID != "sq-1" || result[0].ItemID != "item-1" {
		t.Fatalf("unexpected id fields: %+v", result[0])
	}
	if result[0].Timestamp != "1767090292" {
		t.Fatalf("unexpected timestamp: %+v", result[0])
	}
	if result[0].ConnectionID != "6982771848633227000" {
		t.Fatalf("unexpected connection id: %+v", result[0])
	}
}

func TestSlowQueryRecordsReturnTopRowsAndPreserveSelectedItemAsSourceRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/clinic/api/v1/orgs/org-1/clusters/cluster-9/slowqueries" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("itemID"); got != "item-1" {
			t.Fatalf("unexpected itemID: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"sq-1","digest":"digest-1","query":"select 1","query_time":5.1,"sourceRef":"row-source-1"},
			{"id":"sq-2","digest":"digest-1","query":"select 1","query_time":4.2,"sourceRef":"row-source-2"}
		]`))
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	result, err := client.SlowQueryRecords(context.Background(), SlowQueryRequest{
		Context: RequestContext{
			OrgType:        "tidb-cluster",
			RoutingOrgType: "op",
			OrgID:          "org-1",
			ClusterID:      "cluster-9",
		},
		ItemID:    "item-1",
		StartTime: 1767087000,
		EndTime:   1767090600,
		OrderBy:   "query_time",
		Limit:     100,
	})
	if err != nil {
		t.Fatalf("SlowQueryRecords failed: %v", err)
	}
	if len(result.Records) != 2 {
		t.Fatalf("expected two raw records, got %d", len(result.Records))
	}
	if result.Total != 2 {
		t.Fatalf("expected total=2, got %+v", result)
	}
	if got := result.Records[0].SourceRef; got != "item-1" {
		t.Fatalf("expected row source ref to preserve selected item id, got %+v", result.Records[0])
	}
	if result.Records[0].ExecCount != 1 || result.Records[1].ExecCount != 1 {
		t.Fatalf("expected raw rows to keep exec_count=1, got %+v", result.Records)
	}
}

func TestGetCollectedSlowQueryDetailUsesClusterScopedEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/clinic/api/v1/orgs/org-1/clusters/cluster-9/slowqueries/sq-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("itemID"); got != "item-1" {
			t.Fatalf("unexpected itemID: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sq-1","query":"select 1","digest":"digest-1"}`))
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	result, err := client.GetCollectedSlowQueryDetail(context.Background(), CollectedSlowQueryDetailRequest{
		Context: RequestContext{
			OrgType:        "tidb-cluster",
			RoutingOrgType: "op",
			OrgID:          "org-1",
			ClusterID:      "cluster-9",
		},
		ItemID:      "item-1",
		SlowQueryID: "sq-1",
	})
	if err != nil {
		t.Fatalf("GetCollectedSlowQueryDetail failed: %v", err)
	}
	if got := result["id"]; got != "sq-1" {
		t.Fatalf("unexpected detail payload: %+v", result)
	}
}
