package clinicapi

import (
	"bytes"
	"context"
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

	err = client.EnsureCatalogDataReadable(context.Background(), RequestContext{
		OrgID:     "org-1",
		ClusterID: "cluster-9",
	}, ClinicDataItem{
		ItemID:    "item-1",
		StartTime: 1767087000,
		EndTime:   1767090600,
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

	err = client.EnsureCatalogDataReadable(context.Background(), RequestContext{
		OrgID:     "org-1",
		ClusterID: "cluster-9",
	}, ClinicDataItem{
		ItemID:    "item-1",
		StartTime: 1767087000,
		EndTime:   1767090600,
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

	err = client.EnsureCatalogDataReadable(context.Background(), RequestContext{
		OrgID:     "org-1",
		ClusterID: "cluster-9",
	}, ClinicDataItem{
		ItemID:    "item-1",
		StartTime: 1767087000,
		EndTime:   1767090600,
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

	err = client.EnsureCatalogDataReadable(ctx, RequestContext{
		OrgID:     "org-1",
		ClusterID: "cluster-9",
	}, ClinicDataItem{
		ItemID:    "item-1",
		StartTime: 1767087000,
		EndTime:   1767090600,
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
