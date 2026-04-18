package clinicapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestEnsureCatalogDataReadableAllowsStatus100(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.URL.Query().Get("startTime"); got != "1767087000" {
			t.Fatalf("unexpected startTime: %s", got)
		}
		if got := r.URL.Query().Get("endTime"); got != "1767090600" {
			t.Fatalf("unexpected endTime: %s", got)
		}
		if got := r.URL.Query().Get("data_type"); got != "1" {
			t.Fatalf("unexpected data_type: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"itemID":"item-1","status":100,"startTime":1767087000,"endTime":1767090600,"taskType":1}]}`))
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

	err = client.EnsureCatalogDataReadable(context.Background(), EnsureCatalogDataReadableRequest{
		Context: RequestContext{
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
		Item: ClinicDataItem{
			ItemID:    "item-1",
			StartTime: 1767087000,
			EndTime:   1767090600,
		},
		DataType: CatalogDataTypeRetained,
	})
	if err != nil {
		t.Fatalf("expected status 100 to be readable, got err=%v", err)
	}
}

func TestEnsureCatalogDataReadableTriggersRebuildWhenStatusIsMinusOne(t *testing.T) {
	var rebuildCalls atomic.Int32
	var statusCalls atomic.Int32
	var logs bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status":
			w.Header().Set("Content-Type", "application/json")
			switch statusCalls.Add(1) {
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"itemID":"item-1","status":-1,"startTime":1767087000,"endTime":1767090600,"taskType":1}]}`))
			default:
				_, _ = w.Write([]byte(`{"items":[{"itemID":"item-1","status":100,"startTime":1767087000,"endTime":1767090600,"taskType":1}]}`))
			}
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/rebuild":
			if r.Method != http.MethodPut {
				t.Fatalf("unexpected rebuild method: %s", r.Method)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode rebuild body: %v", err)
			}
			if got := int(payload["dataType"].(float64)); got != 1 {
				t.Fatalf("unexpected rebuild dataType: %v", payload)
			}
			if got := payload["itemID"]; got != "item-1" {
				t.Fatalf("unexpected rebuild itemID: %v", payload)
			}
			rebuildCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:              server.URL,
		BearerToken:          "token",
		Timeout:              5 * time.Second,
		RebuildProbeInterval: time.Millisecond,
		Logger:               log.New(&logs, "", 0),
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	err = client.EnsureCatalogDataReadable(context.Background(), EnsureCatalogDataReadableRequest{
		Context: RequestContext{
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
		Item: ClinicDataItem{
			ItemID:    "item-1",
			StartTime: 1767087000,
			EndTime:   1767090600,
		},
		DataType: CatalogDataTypeRetained,
	})
	if err != nil {
		t.Fatalf("expected rebuild flow to complete, got err=%v", err)
	}
	if rebuildCalls.Load() != 1 {
		t.Fatalf("expected one rebuild call, got %d", rebuildCalls.Load())
	}
	if statusCalls.Load() < 2 {
		t.Fatalf("expected status to be probed more than once, got %d", statusCalls.Load())
	}
	text := logs.String()
	if !strings.Contains(text, "stage=clinic_catalog endpoint=/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status status=rebuild_required") {
		t.Fatalf("expected rebuild_required log, got=%q", text)
	}
	if !strings.Contains(text, "stage=clinic_catalog endpoint=/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status status=rebuild_triggered") {
		t.Fatalf("expected rebuild_triggered log, got=%q", text)
	}
	if !strings.Contains(text, "stage=clinic_catalog endpoint=/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status status=rebuilding") {
		t.Fatalf("expected rebuilding log, got=%q", text)
	}
	if !strings.Contains(text, "stage=clinic_catalog endpoint=/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status status=rebuild_completed") {
		t.Fatalf("expected rebuild_completed log, got=%q", text)
	}
}

func TestEnsureCatalogDataReadableRebuildIncludesCatalogContext(t *testing.T) {
	var rebuildCalls atomic.Int32
	var statusCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status":
			w.Header().Set("Content-Type", "application/json")
			switch statusCalls.Add(1) {
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"itemID":"item-1","status":0,"startTime":1767087000,"endTime":1767090600,"taskType":4}]}`))
			default:
				_, _ = w.Write([]byte(`{"items":[{"itemID":"item-1","status":100,"startTime":1767087000,"endTime":1767090600,"taskType":4}]}`))
			}
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/rebuild":
			if r.Method != http.MethodPut {
				t.Fatalf("unexpected rebuild method: %s", r.Method)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode rebuild body: %v", err)
			}
			if got := int(payload["dataType"].(float64)); got != 4 {
				t.Fatalf("unexpected rebuild dataType: %v", payload)
			}
			if got := payload["itemID"]; got != "item-1" {
				t.Fatalf("unexpected rebuild itemID: %v", payload)
			}
			if _, ok := payload["taskType"]; ok {
				t.Fatalf("expected rebuild taskType to be omitted, got: %v", payload)
			}
			if _, ok := payload["startTime"]; ok {
				t.Fatalf("expected rebuild startTime to be omitted, got: %v", payload)
			}
			if _, ok := payload["endTime"]; ok {
				t.Fatalf("expected rebuild endTime to be omitted, got: %v", payload)
			}
			rebuildCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:              server.URL,
		BearerToken:          "token",
		Timeout:              5 * time.Second,
		RebuildProbeInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	err = client.EnsureCatalogDataReadable(context.Background(), EnsureCatalogDataReadableRequest{
		Context: RequestContext{
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
		Item: ClinicDataItem{
			ItemID:    "item-1",
			StartTime: 1767087000,
			EndTime:   1767090600,
		},
		DataType: CatalogDataTypeCollectedDownload,
	})
	if err != nil {
		t.Fatalf("expected rebuild flow to complete, got err=%v", err)
	}
	if rebuildCalls.Load() != 1 {
		t.Fatalf("expected one rebuild call, got %d", rebuildCalls.Load())
	}
}

func TestEnsureCatalogDataReadableTriggersRebuildForLogsWhenStatusIsZero(t *testing.T) {
	var rebuildCalls atomic.Int32
	var statusCalls atomic.Int32
	var logs bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status":
			if got := r.URL.Query().Get("data_type"); got != "2" {
				t.Fatalf("unexpected data_type: %s", got)
			}
			w.Header().Set("Content-Type", "application/json")
			switch statusCalls.Add(1) {
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"itemID":"item-1","status":0,"startTime":1767087000,"endTime":1767090600,"taskType":2}]}`))
			default:
				_, _ = w.Write([]byte(`{"items":[{"itemID":"item-1","status":100,"startTime":1767087000,"endTime":1767090600,"taskType":2}]}`))
			}
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/rebuild":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode rebuild body: %v", err)
			}
			if got := int(payload["dataType"].(float64)); got != 2 {
				t.Fatalf("unexpected rebuild dataType: %v", payload)
			}
			if got := payload["itemID"]; got != "item-1" {
				t.Fatalf("unexpected rebuild itemID: %v", payload)
			}
			rebuildCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:              server.URL,
		BearerToken:          "token",
		Timeout:              5 * time.Second,
		RebuildProbeInterval: time.Millisecond,
		Logger:               log.New(&logs, "", 0),
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	err = client.EnsureCatalogDataReadable(context.Background(), EnsureCatalogDataReadableRequest{
		Context: RequestContext{
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
		Item: ClinicDataItem{
			ItemID:    "item-1",
			StartTime: 1767087000,
			EndTime:   1767090600,
		},
		DataType: CatalogDataTypeLogs,
	})
	if err != nil {
		t.Fatalf("expected log rebuild flow to complete, got err=%v", err)
	}
	if rebuildCalls.Load() != 1 {
		t.Fatalf("expected one rebuild call, got %d", rebuildCalls.Load())
	}
	if statusCalls.Load() < 2 {
		t.Fatalf("expected multiple probes, got %d", statusCalls.Load())
	}
	text := logs.String()
	if !strings.Contains(text, "status=rebuild_required data_type=2 item_status=0 task_type=2") {
		t.Fatalf("expected log rebuild_required trace, got=%q", text)
	}
	if !strings.Contains(text, "status=rebuild_triggered data_type=2 item_status=0 task_type=2") {
		t.Fatalf("expected log rebuild_triggered trace, got=%q", text)
	}
}

func TestEnsureCatalogDataReadableWaitsForNonReadableStatusWithoutRebuild(t *testing.T) {
	var rebuildCalls atomic.Int32
	var statusCalls atomic.Int32
	var logs bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status":
			w.Header().Set("Content-Type", "application/json")
			switch statusCalls.Add(1) {
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"itemID":"item-1","status":50,"startTime":1767087000,"endTime":1767090600,"taskType":1}]}`))
			default:
				_, _ = w.Write([]byte(`{"items":[{"itemID":"item-1","status":100,"startTime":1767087000,"endTime":1767090600,"taskType":1}]}`))
			}
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/rebuild":
			rebuildCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:              server.URL,
		BearerToken:          "token",
		Timeout:              5 * time.Second,
		RebuildProbeInterval: time.Millisecond,
		Logger:               log.New(&logs, "", 0),
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	err = client.EnsureCatalogDataReadable(context.Background(), EnsureCatalogDataReadableRequest{
		Context: RequestContext{
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
		Item: ClinicDataItem{
			ItemID:    "item-1",
			StartTime: 1767087000,
			EndTime:   1767090600,
		},
		DataType: CatalogDataTypeRetained,
	})
	if err != nil {
		t.Fatalf("expected rebuilding item to become readable, got err=%v", err)
	}
	if rebuildCalls.Load() != 0 {
		t.Fatalf("expected rebuild not to be called, got %d", rebuildCalls.Load())
	}
	if statusCalls.Load() < 2 {
		t.Fatalf("expected multiple probes, got %d", statusCalls.Load())
	}
	text := logs.String()
	if !strings.Contains(text, "stage=clinic_catalog endpoint=/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status status=rebuilding") {
		t.Fatalf("expected rebuilding log, got=%q", text)
	}
	if !strings.Contains(text, "stage=clinic_catalog endpoint=/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status status=rebuild_completed") {
		t.Fatalf("expected rebuild_completed log, got=%q", text)
	}
}

func TestEnsureCatalogDataReadableTimesOutWhileWaitingForRebuild(t *testing.T) {
	var logs bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"itemID":"item-1","status":50,"startTime":1767087000,"endTime":1767090600,"taskType":1}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:              server.URL,
		BearerToken:          "token",
		Timeout:              5 * time.Second,
		RebuildProbeInterval: time.Millisecond,
		Logger:               log.New(&logs, "", 0),
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err = client.EnsureCatalogDataReadable(ctx, EnsureCatalogDataReadableRequest{
		Context: RequestContext{
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
		Item: ClinicDataItem{
			ItemID:    "item-1",
			StartTime: 1767087000,
			EndTime:   1767090600,
		},
		DataType: CatalogDataTypeRetained,
	})
	if err == nil {
		t.Fatalf("expected timeout while waiting for rebuild")
	}
	clinicErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T err=%v", err, err)
	}
	if clinicErr.Class != ErrTimeout {
		t.Fatalf("expected ErrTimeout, got %+v", clinicErr)
	}
	text := logs.String()
	if !strings.Contains(text, "stage=clinic_catalog endpoint=/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status status=rebuild_probe_cancelled") {
		t.Fatalf("expected rebuild_probe_cancelled log, got=%q", text)
	}
}

func TestDownloadCollectedDataUsesClusterScopedEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/clinic/api/v1/orgs/org-1/clusters/cluster-9/download/item-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		_, _ = w.Write([]byte("bundle-bytes"))
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

	body, err := client.DownloadCollectedData(context.Background(), CollectedDataDownloadRequest{
		Context: RequestContext{
			OrgType:        "tidb-cluster",
			RoutingOrgType: "op",
			OrgID:          "org-1",
			ClusterID:      "cluster-9",
		},
		ItemID: "item-1",
	})
	if err != nil {
		t.Fatalf("DownloadCollectedData failed: %v", err)
	}
	if string(body) != "bundle-bytes" {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func TestQuerySlowQueriesUsesSharedRetainedEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/clinic/api/v1/orgs/org-1/clusters/cluster-9/slowqueries" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.URL.Query().Get("itemID"); got != "item-1" {
			t.Fatalf("unexpected itemID: %s", got)
		}
		if got := r.URL.Query().Get("startTime"); got != "1767087000" {
			t.Fatalf("unexpected startTime: %s", got)
		}
		if got := r.URL.Query().Get("endTime"); got != "1767090600" {
			t.Fatalf("unexpected endTime: %s", got)
		}
		if got := r.URL.Query().Get("orderBy"); got != "queryTime" {
			t.Fatalf("unexpected orderBy: %s", got)
		}
		if got := r.URL.Query().Get("desc"); got != "true" {
			t.Fatalf("unexpected desc: %s", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Fatalf("unexpected limit: %s", got)
		}
		if got := r.Header.Get("X-OrgType"); got != "cloud" {
			t.Fatalf("unexpected X-OrgType: %s", got)
		}
		if got := r.Header.Get("X-OrgID"); got != "org-1" {
			t.Fatalf("unexpected X-OrgID: %s", got)
		}
		if got := r.Header.Get("X-ClusterID"); got != "cluster-9" {
			t.Fatalf("unexpected X-ClusterID: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":1,"slowQueries":[{"digest":"digest-1","query":"select * from t","queryTime":1.5,"execCount":3,"user":"root","db":"test","tableNames":["t"],"indexNames":["idx_a"],"sourceRef":"bundle://item-1"}]}`))
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

	result, err := client.QuerySlowQueries(context.Background(), SlowQueryRequest{
		Context: RequestContext{
			OrgType:        "cloud",
			RoutingOrgType: "cloud",
			OrgID:          "org-1",
			ClusterID:      "cluster-9",
		},
		ItemID:    "item-1",
		StartTime: 1767087000,
		EndTime:   1767090600,
		OrderBy:   "queryTime",
		Desc:      true,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("QuerySlowQueries failed: %v", err)
	}
	if result.Total != 1 || len(result.Records) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	record := result.Records[0]
	if record.Digest != "digest-1" || record.SQLText != "select * from t" || record.QueryTime != 1.5 || record.ExecCount != 3 {
		t.Fatalf("unexpected record: %+v", record)
	}
}

func TestQuerySlowQueriesRejectsInvalidRange(t *testing.T) {
	client, err := NewClientWithConfig(Config{
		BaseURL:     "https://example.com",
		BearerToken: "token",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.QuerySlowQueries(context.Background(), SlowQueryRequest{
		Context: RequestContext{
			OrgType:        "op",
			RoutingOrgType: "op",
			OrgID:          "org-1",
			ClusterID:      "cluster-9",
		},
		ItemID:    "item-1",
		StartTime: 1767090600,
		EndTime:   1767087000,
	})
	if err == nil {
		t.Fatalf("expected invalid request error")
	}
	clinicErr, ok := err.(*Error)
	if !ok || clinicErr.Class != ErrInvalidRequest {
		t.Fatalf("expected invalid request clinic error, got %T %v", err, err)
	}
}

func TestQuerySlowQueriesDecodesArrayResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/clinic/api/v1/orgs/org-1/clusters/cluster-9/slowqueries" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"digest":"digest-2","query":"select 1","query_time":2.5,"request_count":7,"user":"root","db":"test","index_names":"idx_a","packageid":"pkg-1"}]`))
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

	result, err := client.QuerySlowQueries(context.Background(), SlowQueryRequest{
		Context: RequestContext{
			OrgType:        "op",
			RoutingOrgType: "op",
			OrgID:          "org-1",
			ClusterID:      "cluster-9",
		},
		ItemID:    "item-1",
		StartTime: 1767087000,
		EndTime:   1767090600,
	})
	if err != nil {
		t.Fatalf("QuerySlowQueries failed: %v", err)
	}
	if result.Total != 1 || len(result.Records) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	record := result.Records[0]
	if record.Digest != "digest-2" || record.SQLText != "select 1" || record.QueryTime != 2.5 || record.ExecCount != 7 || record.SourceRef != "pkg-1" {
		t.Fatalf("unexpected record: %+v", record)
	}
	if len(record.IndexNames) != 1 || record.IndexNames[0] != "idx_a" {
		t.Fatalf("unexpected index names: %+v", record)
	}
}

func TestQuerySlowQuerySamplesInjectsItemScopedSourceRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/clinic/api/v1/orgs/org-1/clusters/cluster-9/slowqueries" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("digest"); got != "digest-1" {
			t.Fatalf("unexpected digest: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":1,"slowQueries":[{"id":"sample-1","digest":"digest-1","query":"select 1","timestamp":"2025-12-30T09:31:00Z","connection_id":"conn-1"}]}`))
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

	result, err := client.QuerySlowQuerySamples(context.Background(), SlowQuerySamplesRequest{
		Context: RequestContext{
			OrgType:        "op",
			RoutingOrgType: "op",
			OrgID:          "org-1",
			ClusterID:      "cluster-9",
		},
		ItemID:    "item-1",
		StartTime: 1767087000,
		EndTime:   1767090600,
		Digest:    "digest-1",
	})
	if err != nil {
		t.Fatalf("QuerySlowQuerySamples failed: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := asTrimmedString(result.Items[0]["source_ref"]); got != "item-1" {
		t.Fatalf("expected source_ref=item-1, got=%q item=%+v", got, result.Items[0])
	}
	if got := asTrimmedString(result.Items[0]["item_id"]); got != "item-1" {
		t.Fatalf("expected item_id=item-1, got=%q item=%+v", got, result.Items[0])
	}
	if got := asTrimmedString(result.Items[0]["connection_id"]); got != "conn-1" {
		t.Fatalf("expected connection_id to remain separate, got=%q item=%+v", got, result.Items[0])
	}
}
