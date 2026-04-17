package cli

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClusterDetailUsesSharedResolveChain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			w.Write([]byte(`{"items":[{"clusterID":"cluster-9","clusterName":"tiup-demo","orgID":"org-tiup","tenantID":"tenant-1","projectID":"","clusterProviderName":"","clusterRegionName":"","clusterDeployType":"tiup-cluster","clusterDeployTypeV2":"tiup-cluster","clusterStatus":"active","clusterCreatedAt":1,"capabilityHints":{}}]}`))
		case "/clinic/api/v1/orgs/org-tiup":
			w.Write([]byte(`{"id":"org-tiup","name":"tiup-org","type":""}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9":
			w.Write([]byte(`{"id":"cluster-9","name":"tiup-demo","topology":{"tidb":3,"tikv":3,"pd":3},"components":null}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	lookup := func(key string) (string, bool) {
		switch key {
		case "CLINIC_API_BASE_URL":
			return server.URL, true
		case "CLINIC_API_KEY":
			return "token", true
		case "CLINIC_CLUSTER_ID":
			return "cluster-9", true
		default:
			return "", false
		}
	}

	var out bytes.Buffer
	err := runClusterDetail(lookup, func() time.Time { return time.Unix(1772777400, 0) }, log.New(io.Discard, "", 0), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "topology=3-tidb / 3-tikv / 3-pd") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestRetainedConfigsGetWaitsForTiUPRebuildCompletion(t *testing.T) {
	var configReads atomic.Int32
	var rebuildCalls atomic.Int32
	var statusCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			w.Write([]byte(`{"items":[{"clusterID":"cluster-9","clusterName":"tiup-demo","orgID":"org-tiup","tenantID":"tenant-1","projectID":"","clusterProviderName":"","clusterRegionName":"","clusterDeployType":"tiup-cluster","clusterDeployTypeV2":"tiup-cluster","clusterStatus":"active","clusterCreatedAt":1,"capabilityHints":{}}]}`))
		case "/clinic/api/v1/orgs/org-tiup":
			w.Write([]byte(`{"id":"org-tiup","name":"tiup-org","type":""}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data":
			w.Write([]byte(`{"total":1,"dataInfos":[{"startTime":1767087000,"endTime":1767090600,"itemID":"item-1","filename":"bundle.zip","collectors":["log.slow"],"haveLog":true,"haveMetric":true,"haveConfig":true}]}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data_status":
			if got := r.URL.Query().Get("startTime"); got != "1767087000" {
				t.Fatalf("unexpected startTime: %s", got)
			}
			if got := r.URL.Query().Get("endTime"); got != "1767090600" {
				t.Fatalf("unexpected endTime: %s", got)
			}
			if got := r.URL.Query().Get("data_type"); got != "1" {
				t.Fatalf("unexpected data_type: %s", got)
			}
			switch statusCalls.Add(1) {
			case 1:
				w.Write([]byte(`{"items":[{"itemID":"item-1","status":-1,"startTime":1767087000,"endTime":1767090600,"taskType":1}]}`))
			default:
				w.Write([]byte(`{"items":[{"itemID":"item-1","status":100,"startTime":1767087000,"endTime":1767090600,"taskType":1}]}`))
			}
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/rebuild":
			if r.Method != http.MethodPut {
				t.Fatalf("unexpected rebuild method: %s", r.Method)
			}
			rebuildCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		case "/clinic/api/v1/data/config":
			configReads.Add(1)
			w.Write([]byte(`{"configs":[{"component":"tikv","key":"storage.block-cache.capacity","value":"8GiB","sourceRef":"config.json"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	lookup := func(key string) (string, bool) {
		switch key {
		case "CLINIC_API_BASE_URL":
			return server.URL, true
		case "CLINIC_API_KEY":
			return "token", true
		case "CLINIC_CLUSTER_ID":
			return "cluster-9", true
		case "CLINIC_REBUILD_PROBE_INTERVAL":
			return "1ms", true
		default:
			return "", false
		}
	}

	var out bytes.Buffer
	err := runRetainedConfigsGet(lookup, func() time.Time { return time.Unix(1772777400, 0) }, log.New(io.Discard, "", 0), &out)
	if err != nil {
		t.Fatalf("expected retained configs read to succeed after rebuild, got %v", err)
	}
	if rebuildCalls.Load() != 1 {
		t.Fatalf("expected rebuild to be triggered once, got %d", rebuildCalls.Load())
	}
	if statusCalls.Load() < 2 {
		t.Fatalf("expected multiple data_status probes, got %d", statusCalls.Load())
	}
	if configReads.Load() != 1 {
		t.Fatalf("expected config endpoint to be read once, got %d reads", configReads.Load())
	}
	if !strings.Contains(out.String(), "total=1") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
