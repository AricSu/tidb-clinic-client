package clinicapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQueryCloudSlowQueriesUsesNGMEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ngm/api/v1/slow_query/list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("begin_time"); got != "1767087000" {
			t.Fatalf("unexpected begin_time: %s", got)
		}
		if got := r.URL.Query().Get("end_time"); got != "1767090600" {
			t.Fatalf("unexpected end_time: %s", got)
		}
		if got := r.URL.Query().Get("orderBy"); got != "timestamp" {
			t.Fatalf("unexpected orderBy: %s", got)
		}
		if got := r.URL.Query().Get("desc"); got != "true" {
			t.Fatalf("unexpected desc: %s", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Fatalf("unexpected limit: %s", got)
		}
		if got := r.URL.Query().Get("fields"); got != "query,timestamp,query_time,memory_max,request_count,connection_id" {
			t.Fatalf("unexpected fields: %s", got)
		}
		if got := r.URL.Query().Get("show_internal"); got != "false" {
			t.Fatalf("unexpected show_internal: %s", got)
		}
		if got := r.Header.Get("X-Provider"); got != "aws" {
			t.Fatalf("unexpected X-Provider: %s", got)
		}
		if got := r.Header.Get("X-Project-Id"); got != "project-1" {
			t.Fatalf("unexpected X-Project-Id: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"digest":"digest-1","query":"select 1","query_time":1.5,"db":"test","connection_id":"conn-1"}]`))
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

	result, err := client.QueryCloudSlowQueries(context.Background(), CloudSlowQueryRequest{
		Target: CloudNGMTarget{
			Provider:   "aws",
			Region:     "eu-west-1",
			TenantID:   "tenant-1",
			ProjectID:  "project-1",
			ClusterID:  "cluster-9",
			DeployType: "dedicated",
		},
		BeginTime:    1767087000,
		EndTime:      1767090600,
		OrderBy:      "timestamp",
		Desc:         true,
		Limit:        10,
		ShowInternal: false,
	})
	if err != nil {
		t.Fatalf("QueryCloudSlowQueries failed: %v", err)
	}
	if result.Total != 1 || len(result.Records) != 1 || result.Records[0].Digest != "digest-1" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestQueryCloudSlowQuerySamplesUsesNGMEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ngm/api/v1/slow_query/list":
			if got := r.URL.Query().Get("digest"); got != "digest-1" {
				t.Fatalf("unexpected digest: %s", got)
			}
			if got := r.URL.Query().Get("fields"); got != "query,timestamp,query_time,memory_max,connection_id" {
				t.Fatalf("unexpected fields: %s", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"digest":"digest-1","query":"select 1","timestamp":1776479170.793401,"query_time":7.080122851,"memory_max":1024,"connection_id":"3977388254"}]`))
		case "/ngm/api/v1/slow_query/detail":
			if got := r.URL.Query().Get("connect_id"); got != "3977388254" {
				t.Fatalf("unexpected connect_id: %s", got)
			}
			if got := r.URL.Query().Get("digest"); got != "digest-1" {
				t.Fatalf("unexpected detail digest: %s", got)
			}
			if got := r.URL.Query().Get("timestamp"); got != "1776479170.793401" {
				t.Fatalf("unexpected detail timestamp: %s", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"digest":"digest-1","query":"select 1","timestamp":1776479170.793401,"query_time":7.080122851,"memory_max":1024,"connection_id":"3977388254","plan":"plan-body","db":"test"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
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

	result, err := client.QueryCloudSlowQuerySamples(context.Background(), CloudSlowQueryRequest{
		Target: CloudNGMTarget{
			Provider:   "aws",
			Region:     "eu-west-1",
			TenantID:   "tenant-1",
			ProjectID:  "project-1",
			ClusterID:  "cluster-9",
			DeployType: "dedicated",
		},
		BeginTime:    1767087000,
		EndTime:      1767090600,
		OrderBy:      "timestamp",
		Digest:       "digest-1",
		Fields:       []string{"query", "timestamp", "query_time", "memory_max"},
		Limit:        100,
		ShowInternal: false,
	})
	if err != nil {
		t.Fatalf("QueryCloudSlowQuerySamples failed: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := asTrimmedString(result.Items[0]["connection_id"]); got != "3977388254" {
		t.Fatalf("unexpected connection_id: %q item=%+v", got, result.Items[0])
	}
	if got := asTrimmedString(result.Items[0]["plan"]); got != "plan-body" {
		t.Fatalf("expected detail payload to be merged, got=%q item=%+v", got, result.Items[0])
	}
	if got := asTrimmedString(result.Items[0]["source_ref"]); got != "" {
		t.Fatalf("expected no source_ref injection for cloud samples, got=%q item=%+v", got, result.Items[0])
	}
}
