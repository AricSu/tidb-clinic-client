package clinic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
	"github.com/AricSu/tidb-clinic-client/internal/model"
)

func TestClinicServiceQuerySlowQueriesUsesLogReadableDataType(t *testing.T) {
	var rebuildCalls atomic.Int32
	var statusCalls atomic.Int32
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
			if _, ok := payload["taskType"]; ok {
				t.Fatalf("expected rebuild taskType to be omitted, got: %v", payload)
			}
			rebuildCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/slowqueries":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"total":0,"slowQueries":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	apiClient, err := apitypes.NewClientWithConfig(apitypes.Config{
		BaseURL:              server.URL,
		BearerToken:          "token",
		Timeout:              5 * time.Second,
		RebuildProbeInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	service := &clinicServiceClient{api: apiClient, client: &Client{cfg: model.Config{RebuildProbeInterval: time.Millisecond}}}
	_, err = service.QuerySlowQueries(context.Background(), apitypes.RequestContext{
		OrgType:        "tidb-cluster",
		RoutingOrgType: "op",
		OrgID:          "org-1",
		ClusterID:      "cluster-9",
	}, apitypes.ClinicDataItem{
		ItemID:    "item-1",
		StartTime: 1767087000,
		EndTime:   1767090600,
	}, SlowQueryQuery{
		Start: 1767087000,
		End:   1767090600,
	})
	if err != nil {
		t.Fatalf("QuerySlowQueries failed: %v", err)
	}
	if rebuildCalls.Load() != 1 {
		t.Fatalf("expected one rebuild call, got %d", rebuildCalls.Load())
	}
	if statusCalls.Load() < 2 {
		t.Fatalf("expected multiple status probes, got %d", statusCalls.Load())
	}
}

func TestClinicServiceQuerySlowQueriesWaitsWhileLogProcessing(t *testing.T) {
	var queryCalls atomic.Int32
	var statusCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status":
			statusCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"itemID":"item-1","status":100,"startTime":1767087000,"endTime":1767090600,"taskType":1}]}`))
		case "/ngm/api/v1/slow_query/list":
			if queryCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":true,"message":"the log is processing","code":"common.bad_request"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"digest":"digest-1","query":"select * from t","query_time":1.5,"exec_count":3,"user":"root","db":"test"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	apiClient, err := apitypes.NewClientWithConfig(apitypes.Config{
		BaseURL:              server.URL,
		BearerToken:          "token",
		Timeout:              5 * time.Second,
		RebuildProbeInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	service := &clinicServiceClient{api: apiClient, client: &Client{cfg: model.Config{RebuildProbeInterval: time.Millisecond}}}
	result, err := service.QueryCloudSlowQueries(context.Background(), apitypes.CloudNGMTarget{
		Provider:   "aws",
		Region:     "eu-west-1",
		TenantID:   "tenant-1",
		ProjectID:  "project-1",
		ClusterID:  "cluster-9",
		DeployType: "dedicated",
	}, SlowQueryQuery{
		Start: 1767087000,
		End:   1767090600,
	})
	if err != nil {
		t.Fatalf("QuerySlowQueries failed: %v", err)
	}
	if statusCalls.Load() != 0 {
		t.Fatalf("expected cloud slowqueries to skip data_status probes, got %d", statusCalls.Load())
	}
	if queryCalls.Load() < 2 {
		t.Fatalf("expected slowqueries to be retried, got %d calls", queryCalls.Load())
	}
	if result.Total != 1 || len(result.Records) != 1 || result.Records[0].Digest != "digest-1" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestClinicServiceQuerySlowQuerySamplesUsesLogReadableDataType(t *testing.T) {
	var rebuildCalls atomic.Int32
	var statusCalls atomic.Int32
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
			rebuildCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/slowqueries":
			if got := r.URL.Query().Get("digest"); got != "digest-1" {
				t.Fatalf("unexpected digest: %s", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"total":1,"slowQueries":[{"id":"sample-1","digest":"digest-1","query":"select * from t","timestamp":"2025-12-30T09:31:00Z","connection_id":"conn-1"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	apiClient, err := apitypes.NewClientWithConfig(apitypes.Config{
		BaseURL:              server.URL,
		BearerToken:          "token",
		Timeout:              5 * time.Second,
		RebuildProbeInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	service := &clinicServiceClient{api: apiClient, client: &Client{cfg: model.Config{RebuildProbeInterval: time.Millisecond}}}
	result, err := service.QuerySlowQuerySamples(context.Background(), apitypes.RequestContext{
		OrgType:        "tidb-cluster",
		RoutingOrgType: "op",
		OrgID:          "org-1",
		ClusterID:      "cluster-9",
	}, apitypes.ClinicDataItem{
		ItemID:    "item-1",
		StartTime: 1767087000,
		EndTime:   1767090600,
	}, SlowQuerySamplesQuery{
		Digest: "digest-1",
		Start:  "2025-12-30T09:30:00Z",
		End:    "2025-12-30T10:30:00Z",
	}, 1767087000, 1767090600)
	if err != nil {
		t.Fatalf("QuerySlowQuerySamples failed: %v", err)
	}
	if rebuildCalls.Load() != 1 {
		t.Fatalf("expected one rebuild call, got %d", rebuildCalls.Load())
	}
	if statusCalls.Load() < 2 {
		t.Fatalf("expected multiple status probes, got %d", statusCalls.Load())
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := stringifyAny(result.Items[0]["source_ref"]); got != "item-1" {
		t.Fatalf("expected item-scoped source_ref, got=%q item=%+v", got, result.Items[0])
	}
}

func TestClinicServiceQuerySlowQuerySamplesCloudSkipsCatalogReadiness(t *testing.T) {
	var statusCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/data_status":
			statusCalls.Add(1)
			http.Error(w, "cloud slow query samples should not probe data status", http.StatusInternalServerError)
		case "/ngm/api/v1/slow_query/list":
			if got := r.URL.Query().Get("digest"); got != "digest-1" {
				t.Fatalf("unexpected digest: %s", got)
			}
			if got := r.URL.Query().Get("fields"); got != "query,timestamp,connection_id" {
				t.Fatalf("unexpected fields: %s", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"digest":"digest-1","query":"select * from t","timestamp":1776479170.793401,"connection_id":"conn-1"}]`))
		case "/ngm/api/v1/slow_query/detail":
			if got := r.URL.Query().Get("connect_id"); got != "conn-1" {
				t.Fatalf("unexpected connect_id: %s", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"digest":"digest-1","query":"select * from t","timestamp":1776479170.793401,"connection_id":"conn-1","plan":"detail-plan"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	apiClient, err := apitypes.NewClientWithConfig(apitypes.Config{
		BaseURL:              server.URL,
		BearerToken:          "token",
		Timeout:              5 * time.Second,
		RebuildProbeInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	service := &clinicServiceClient{api: apiClient, client: &Client{cfg: model.Config{RebuildProbeInterval: time.Millisecond}}}
	result, err := service.QueryCloudSlowQuerySamples(context.Background(), apitypes.CloudNGMTarget{
		Provider:   "aws",
		Region:     "eu-west-1",
		TenantID:   "tenant-1",
		ProjectID:  "project-1",
		ClusterID:  "cluster-9",
		DeployType: "dedicated",
	}, SlowQuerySamplesQuery{
		Digest: "digest-1",
		Start:  "2025-12-30T09:30:00Z",
		End:    "2025-12-30T10:30:00Z",
		Fields: []string{"query", "timestamp"},
	}, 1767087000, 1767090600)
	if err != nil {
		t.Fatalf("QuerySlowQuerySamples failed: %v", err)
	}
	if statusCalls.Load() != 0 {
		t.Fatalf("expected cloud slow query samples to skip data_status probes, got %d", statusCalls.Load())
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := stringifyAny(result.Items[0]["plan"]); got != "detail-plan" {
		t.Fatalf("expected cloud slow query samples to include detail payload, got=%q item=%+v", got, result.Items[0])
	}
	if got := stringifyAny(result.Items[0]["source_ref"]); got != "" {
		t.Fatalf("expected cloud slow query samples to keep source_ref untouched when item is absent, got=%q item=%+v", got, result.Items[0])
	}
}
