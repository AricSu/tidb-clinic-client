package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	clinicapi "github.com/AricSu/tidb-clinic-client"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testPortalURL(baseURL, orgID, clusterID string, from, to int64) string {
	url := fmt.Sprintf("%s/portal/#/orgs/%s/clusters/%s", strings.TrimRight(baseURL, "/"), orgID, clusterID)
	switch {
	case from > 0 && to > 0:
		return fmt.Sprintf("%s?from=%d&to=%d", url, from, to)
	case from > 0:
		return fmt.Sprintf("%s?from=%d", url, from)
	case to > 0:
		return fmt.Sprintf("%s?to=%d", url, to)
	default:
		return url
	}
}

func testLookupWithPortal(portalURL string, values ...string) func(string) (string, bool) {
	entries := map[string]string{
		"CLINIC_PORTAL_URL": portalURL,
		"CLINIC_API_KEY":    "token",
	}
	for i := 0; i+1 < len(values); i += 2 {
		entries[values[i]] = values[i+1]
	}
	return func(key string) (string, bool) {
		value, ok := entries[key]
		return value, ok
	}
}

func TestClusterInfoPrintsRawClusterDetailJSON(t *testing.T) {
	var searchCalls atomic.Int32
	var detailCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			searchCalls.Add(1)
			http.Error(w, "clusters search should not be called", http.StatusInternalServerError)
		case "/clinic/api/v1/orgs/1372813089196930348/clusters/7372714695339837431":
			detailCalls.Add(1)
			w.Write([]byte(`{
				"id":"7372714695339837431",
				"name":"production-tidb-gold-1",
				"clusterType":"tidb-cluster",
				"status":"active",
				"deployType":"tiup-cluster",
				"deployTypeV2":"tiup-cluster",
				"topology":{"tidb":49,"tikv":30,"pd":3}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	lookup := testLookupWithPortal(
		testPortalURL(server.URL, "1372813089196930348", "7372714695339837431", 1773547800, 1773548100),
	)

	var out bytes.Buffer
	err := runClusterInfo(lookup, func() time.Time { return time.Unix(1772777400, 0) }, log.New(io.Discard, "", 0), &out)
	if err != nil {
		t.Fatalf("expected cluster info command to print raw detail json, got %v", err)
	}
	if searchCalls.Load() != 0 {
		t.Fatalf("expected dashboard search endpoint not to be called, got %d", searchCalls.Load())
	}
	if detailCalls.Load() != 1 {
		t.Fatalf("expected detail endpoint to be called once, got %d", detailCalls.Load())
	}
	text := out.String()
	if !strings.Contains(text, `"id": "7372714695339837431"`) {
		t.Fatalf("expected raw json id, got=%q", text)
	}
	if !strings.Contains(text, `"deployTypeV2": "tiup-cluster"`) {
		t.Fatalf("expected raw json deployTypeV2, got=%q", text)
	}
	if strings.Contains(text, "clusters=1") {
		t.Fatalf("expected legacy formatted search output to be removed, got=%q", text)
	}
}

func TestClusterEventListPassesFiltersToActivityHub(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			w.Write([]byte(`{"items":[{"clusterID":"cluster-9","clusterName":"cloud-demo","orgID":"org-cloud","tenantID":"tenant-1","projectID":"project-1","clusterProviderName":"aws","clusterRegionName":"eu-west-1","clusterDeployType":"dedicated","clusterDeployTypeV2":"dedicated","clusterStatus":"active","clusterCreatedAt":1,"capabilityHints":{}}]}`))
		case "/clinic/api/v1/orgs/org-cloud":
			w.Write([]byte(`{"id":"org-cloud","name":"cloud-org","type":"cloud"}`))
		case "/clinic/api/v1/activityhub/applications/org-cloud/targets/cluster-9/activities":
			if got := r.URL.Query().Get("begin_ts"); got != "1773547800" {
				t.Fatalf("unexpected begin_ts: %s", got)
			}
			if got := r.URL.Query().Get("end_ts"); got != "1773548100" {
				t.Fatalf("unexpected end_ts: %s", got)
			}
			if got := r.URL.Query().Get("name"); got != "backup" {
				t.Fatalf("unexpected name: %s", got)
			}
			if got := r.URL.Query().Get("severity"); got != "3" {
				t.Fatalf("unexpected severity: %s", got)
			}
			w.Write([]byte(`{
				"activities":[{
					"activity_id":"event-1",
					"name":"backupcluster",
					"display_name":"BackupCluster",
					"calibration_time":1776431041,
					"payload":{"summary":"ok"}
				}]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	lookup := testLookupWithPortal(
		testPortalURL(server.URL, "org-cloud", "cluster-9", 1773547800, 1773548100),
		"CLINIC_EVENT_NAME", "backup",
		"CLINIC_EVENT_SEVERITY", "critical",
	)

	var out bytes.Buffer
	err := runCloudEventsQuery(lookup, func() time.Time { return time.Unix(1772777400, 0) }, log.New(io.Discard, "", 0), &out)
	if err != nil {
		t.Fatalf("expected cluster event list to succeed, got %v", err)
	}
	text := out.String()
	if !strings.Contains(text, `"activity_id": "event-1"`) {
		t.Fatalf("unexpected output: %s", text)
	}
	if !strings.Contains(text, `"name": "backupcluster"`) {
		t.Fatalf("unexpected event output: %s", text)
	}
	if !strings.Contains(text, `"display_name": "BackupCluster"`) {
		t.Fatalf("unexpected event output: %s", text)
	}
	if strings.Contains(text, "event[0] id=") {
		t.Fatalf("unexpected event output: %s", text)
	}
}

func TestClusterEventListRejectsTiUPClusterTarget(t *testing.T) {
	var activityCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			w.Write([]byte(`{"items":[{"clusterID":"cluster-9","clusterName":"tiup-demo","orgID":"org-tiup","tenantID":"tenant-1","projectID":"","clusterProviderName":"","clusterRegionName":"","clusterDeployType":"tiup-cluster","clusterDeployTypeV2":"tiup-cluster","clusterStatus":"active","clusterCreatedAt":1,"capabilityHints":{}}]}`))
		case "/clinic/api/v1/orgs/org-tiup":
			w.Write([]byte(`{"id":"org-tiup","name":"tiup-org","type":"op"}`))
		case "/clinic/api/v1/activityhub/applications/org-tiup/targets/cluster-9/activities":
			activityCalls.Add(1)
			http.Error(w, "events should not be requested for tiup-cluster", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	lookup := testLookupWithPortal(
		testPortalURL(server.URL, "org-tiup", "cluster-9", 1773547800, 1773548100),
	)

	var out bytes.Buffer
	err := runCloudEventsQuery(lookup, func() time.Time { return time.Unix(1772777400, 0) }, log.New(io.Discard, "", 0), &out)
	if err == nil {
		t.Fatalf("expected tiup-cluster events to be rejected")
	}
	if got := clinicapi.ClassOf(err); got != clinicapi.ErrUnsupported {
		t.Fatalf("expected unsupported error class, got %q (%v)", got, err)
	}
	if !strings.Contains(err.Error(), "non-cloud deployments are not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
	if activityCalls.Load() != 0 {
		t.Fatalf("expected no activityhub request for tiup-cluster, got %d", activityCalls.Load())
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output on unsupported tiup-cluster events, got %q", out.String())
	}
}

func TestSlowQueryCloudSkipsCatalogItemSelection(t *testing.T) {
	var slowQueryCalls atomic.Int32
	var catalogCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			w.Write([]byte(`{"items":[{"clusterID":"cluster-9","clusterName":"cloud-demo","orgID":"org-cloud","tenantID":"tenant-1","projectID":"project-1","clusterProviderName":"aws","clusterRegionName":"eu-west-1","clusterDeployType":"dedicated","clusterDeployTypeV2":"dedicated","clusterStatus":"active","clusterCreatedAt":1,"capabilityHints":{}}]}`))
		case "/clinic/api/v1/orgs/org-cloud":
			w.Write([]byte(`{"id":"org-cloud","name":"cloud-org","type":"cloud"}`))
		case "/clinic/api/v1/orgs/org-cloud/clusters/cluster-9/data":
			catalogCalls.Add(1)
			http.Error(w, "cloud slowquery should not list catalog data", http.StatusInternalServerError)
		case "/ngm/api/v1/slow_query/list":
			slowQueryCalls.Add(1)
			if got := r.URL.Query().Get("begin_time"); got != "1773547800" {
				t.Fatalf("unexpected begin_time: %s", got)
			}
			if got := r.URL.Query().Get("end_time"); got != "1773548100" {
				t.Fatalf("unexpected end_time: %s", got)
			}
			if got := r.URL.Query().Get("fields"); got != "query,timestamp,query_time,memory_max,request_count,connection_id" {
				t.Fatalf("unexpected fields: %s", got)
			}
			w.Write([]byte(`[{"digest":"digest-1","query":"select 1","query_time":1.5,"exec_count":3}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	lookup := testLookupWithPortal(
		testPortalURL(server.URL, "org-cloud", "cluster-9", 1773547800, 1773548100),
	)

	var out bytes.Buffer
	err := runSlowQuery(lookup, func() time.Time { return time.Unix(1772777400, 0) }, log.New(io.Discard, "", 0), &out)
	if err != nil {
		t.Fatalf("expected cloud slowquery to succeed, got %v", err)
	}
	if catalogCalls.Load() != 0 {
		t.Fatalf("expected no catalog selection for cloud slowquery, got %d", catalogCalls.Load())
	}
	if slowQueryCalls.Load() != 1 {
		t.Fatalf("expected one slowquery request, got %d", slowQueryCalls.Load())
	}
	if !strings.Contains(out.String(), `"digest": "digest-1"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestSlowQuerySamplesCloudSkipsCatalogItemSelection(t *testing.T) {
	var slowQueryCalls atomic.Int32
	var catalogCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			w.Write([]byte(`{"items":[{"clusterID":"cluster-9","clusterName":"cloud-demo","orgID":"org-cloud","tenantID":"tenant-1","projectID":"project-1","clusterProviderName":"aws","clusterRegionName":"eu-west-1","clusterDeployType":"dedicated","clusterDeployTypeV2":"dedicated","clusterStatus":"active","clusterCreatedAt":1,"capabilityHints":{}}]}`))
		case "/clinic/api/v1/orgs/org-cloud":
			w.Write([]byte(`{"id":"org-cloud","name":"cloud-org","type":"cloud"}`))
		case "/clinic/api/v1/orgs/org-cloud/clusters/cluster-9/data":
			catalogCalls.Add(1)
			http.Error(w, "cloud slowquery samples should not list catalog data", http.StatusInternalServerError)
		case "/ngm/api/v1/slow_query/list":
			slowQueryCalls.Add(1)
			if got := r.URL.Query().Get("digest"); got != "digest-1" {
				t.Fatalf("unexpected digest: %s", got)
			}
			w.Write([]byte(`[{"id":"sample-1","digest":"digest-1","query":"select 1","timestamp":"2025-12-30T09:31:00Z","connection_id":"conn-1"}]`))
		case "/ngm/api/v1/slow_query/detail":
			if got := r.URL.Query().Get("connect_id"); got != "conn-1" {
				t.Fatalf("unexpected connect_id: %s", got)
			}
			w.Write([]byte(`{"id":"sample-1","digest":"digest-1","query":"select 1","timestamp":"2025-12-30T09:31:00Z","connection_id":"conn-1","plan":"detail-plan"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	lookup := testLookupWithPortal(
		testPortalURL(server.URL, "org-cloud", "cluster-9", 1767087000, 1767090600),
		"CLINIC_SLOWQUERY_DIGEST", "digest-1",
	)

	var out bytes.Buffer
	err := runSlowQuery(lookup, func() time.Time { return time.Unix(1772777400, 0) }, log.New(io.Discard, "", 0), &out)
	if err != nil {
		t.Fatalf("expected cloud slowquery samples to succeed, got %v", err)
	}
	if catalogCalls.Load() != 0 {
		t.Fatalf("expected no catalog selection for cloud slowquery samples, got %d", catalogCalls.Load())
	}
	if slowQueryCalls.Load() != 1 {
		t.Fatalf("expected one slowquery request, got %d", slowQueryCalls.Load())
	}
	if !strings.Contains(out.String(), `"connection_id": "conn-1"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
	if !strings.Contains(out.String(), `"plan": "detail-plan"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestCollectedDataDownloadReadsTiUPBundle(t *testing.T) {
	var downloadCalls atomic.Int32
	var statusCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			w.Write([]byte(`{"items":[{"clusterID":"cluster-9","clusterName":"tiup-demo","orgID":"org-tiup","tenantID":"tenant-1","projectID":"","clusterProviderName":"","clusterRegionName":"","clusterDeployType":"tiup-cluster","clusterDeployTypeV2":"tiup-cluster","clusterStatus":"active","clusterCreatedAt":1,"capabilityHints":{}}]}`))
		case "/clinic/api/v1/orgs/org-tiup":
			w.Write([]byte(`{"id":"org-tiup","name":"tiup-org","type":""}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data":
			w.Write([]byte(`{"total":1,"dataInfos":[{"startTime":1767677814,"endTime":1767684186,"itemID":"item-1","filename":"bundle.zip","collectors":["log.std"],"haveLog":true,"haveMetric":true,"haveConfig":true}]}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data_status":
			if got := r.URL.Query().Get("data_type"); got != "4" {
				t.Fatalf("unexpected data_type: %s", got)
			}
			statusCalls.Add(1)
			w.Write([]byte(`{"items":[{"itemID":"item-1","status":100,"startTime":1767677814,"endTime":1767684186,"taskType":4}]}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/download/item-1":
			downloadCalls.Add(1)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("bundle-content"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "bundle.zip")
	lookup := testLookupWithPortal(
		testPortalURL(server.URL, "org-tiup", "cluster-9", 1767677814, 1767684186),
		"CLINIC_OUTPUT_PATH", outputPath,
		"CLINIC_REBUILD_PROBE_INTERVAL", "20ms",
	)

	var out bytes.Buffer
	err := runCollectedDataDownload(lookup, func() time.Time { return time.Unix(1772777400, 0) }, log.New(io.Discard, "", 0), &out)
	if err != nil {
		t.Fatalf("expected collected data download to succeed, got %v", err)
	}
	if statusCalls.Load() != 1 {
		t.Fatalf("expected one data_status probe, got %d", statusCalls.Load())
	}
	if downloadCalls.Load() != 1 {
		t.Fatalf("expected one download call, got %d", downloadCalls.Load())
	}
	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(body) != "bundle-content" {
		t.Fatalf("unexpected file content: %q", string(body))
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("expected json output, got err=%v output=%s", err, out.String())
	}
	if got := payload["path"]; got != outputPath {
		t.Fatalf("expected path %q, got %#v", outputPath, got)
	}
}

func TestCollectedDataDownloadWritesCliProgressMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			w.Write([]byte(`{"items":[{"clusterID":"cluster-9","clusterName":"tiup-demo","orgID":"org-tiup","tenantID":"tenant-1","projectID":"","clusterProviderName":"","clusterRegionName":"","clusterDeployType":"tiup-cluster","clusterDeployTypeV2":"tiup-cluster","clusterStatus":"active","clusterCreatedAt":1,"capabilityHints":{}}]}`))
		case "/clinic/api/v1/orgs/org-tiup":
			w.Write([]byte(`{"id":"org-tiup","name":"tiup-org","type":""}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data":
			w.Write([]byte(`{"total":1,"dataInfos":[{"startTime":1767677814,"endTime":1767684186,"itemID":"item-1","filename":"bundle.zip","collectors":["log.std"],"haveLog":true,"haveMetric":true,"haveConfig":true}]}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data_status":
			w.Write([]byte(`{"items":[{"itemID":"item-1","status":100,"startTime":1767677814,"endTime":1767684186,"taskType":4}]}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/download/item-1":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("bundle-content"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "bundle.zip")
	lookup := testLookupWithPortal(
		testPortalURL(server.URL, "org-tiup", "cluster-9", 1767677814, 1767684186),
		"CLINIC_OUTPUT_PATH", outputPath,
	)

	var progress bytes.Buffer
	var out bytes.Buffer
	err := runCollectedDataDownload(lookup, func() time.Time { return time.Unix(1772777400, 0) }, log.New(&progress, "", 0), &out)
	if err != nil {
		t.Fatalf("expected collected data download to succeed, got %v", err)
	}
	text := progress.String()
	if !strings.Contains(text, "正在选择 collected data bundle...") {
		t.Fatalf("expected selection progress, got=%q", text)
	}
	if !strings.Contains(text, "正在检查 bundle 状态...") {
		t.Fatalf("expected status progress, got=%q", text)
	}
	if !strings.Contains(text, "正在下载 bundle...") {
		t.Fatalf("expected download progress, got=%q", text)
	}
	if !strings.Contains(text, "bundle 下载完成（") {
		t.Fatalf("expected completion progress, got=%q", text)
	}
	if strings.Contains(text, "stage=clinic_api") {
		t.Fatalf("expected raw request logs to be removed, got=%q", text)
	}
}

func TestCollectedDataDownloadAutoSelectsLatestItemWhenItemIDMissing(t *testing.T) {
	var downloadCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			w.Write([]byte(`{"items":[{"clusterID":"cluster-9","clusterName":"tiup-demo","orgID":"org-tiup","tenantID":"tenant-1","projectID":"","clusterProviderName":"","clusterRegionName":"","clusterDeployType":"tiup-cluster","clusterDeployTypeV2":"tiup-cluster","clusterStatus":"active","clusterCreatedAt":1,"capabilityHints":{}}]}`))
		case "/clinic/api/v1/orgs/org-tiup":
			w.Write([]byte(`{"id":"org-tiup","name":"tiup-org","type":""}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data":
			w.Write([]byte(`{"total":2,"dataInfos":[{"startTime":1767677814,"endTime":1767680186,"itemID":"item-old","filename":"old.zip","collectors":["log.std"],"haveLog":true,"haveMetric":true,"haveConfig":true},{"startTime":1767679200,"endTime":1767682800,"itemID":"item-new","filename":"new.zip","collectors":["log.std"],"haveLog":true,"haveMetric":true,"haveConfig":true}]}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data_status":
			if got := r.URL.Query().Get("startTime"); got != "1767679200" {
				t.Fatalf("unexpected selected startTime: %s", got)
			}
			if got := r.URL.Query().Get("endTime"); got != "1767682800" {
				t.Fatalf("unexpected selected endTime: %s", got)
			}
			if got := r.URL.Query().Get("data_type"); got != "4" {
				t.Fatalf("unexpected data_type: %s", got)
			}
			w.Write([]byte(`{"items":[{"itemID":"item-new","status":100,"startTime":1767679200,"endTime":1767682800,"taskType":4}]}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/download/item-new":
			downloadCalls.Add(1)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("new-bundle"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "bundle.zip")
	lookup := testLookupWithPortal(
		testPortalURL(server.URL, "org-tiup", "cluster-9", 1767683400-int64(10*time.Minute/time.Second), 1767683400),
		"CLINIC_OUTPUT_PATH", outputPath,
		"CLINIC_REBUILD_PROBE_INTERVAL", "20ms",
	)

	var out bytes.Buffer
	err := runCollectedDataDownload(lookup, func() time.Time { return time.Unix(1767683400, 0) }, log.New(io.Discard, "", 0), &out)
	if err != nil {
		t.Fatalf("expected auto-selected collected data download to succeed, got %v", err)
	}
	if downloadCalls.Load() != 1 {
		t.Fatalf("expected one download call, got %d", downloadCalls.Load())
	}
	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(body) != "new-bundle" {
		t.Fatalf("unexpected file content: %q", string(body))
	}
}

func TestCollectedDataDownloadEmitsPeriodicDownloadReminder(t *testing.T) {
	previousInterval := progressReminderInterval
	progressReminderInterval = 5 * time.Millisecond
	defer func() { progressReminderInterval = previousInterval }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			w.Write([]byte(`{"items":[{"clusterID":"cluster-9","clusterName":"tiup-demo","orgID":"org-tiup","tenantID":"tenant-1","projectID":"","clusterProviderName":"","clusterRegionName":"","clusterDeployType":"tiup-cluster","clusterDeployTypeV2":"tiup-cluster","clusterStatus":"active","clusterCreatedAt":1,"capabilityHints":{}}]}`))
		case "/clinic/api/v1/orgs/org-tiup":
			w.Write([]byte(`{"id":"org-tiup","name":"tiup-org","type":""}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data":
			w.Write([]byte(`{"total":1,"dataInfos":[{"startTime":1767677814,"endTime":1767684186,"itemID":"item-1","filename":"bundle.zip","collectors":["log.std"],"haveLog":true,"haveMetric":true,"haveConfig":true}]}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data_status":
			w.Write([]byte(`{"items":[{"itemID":"item-1","status":100,"startTime":1767677814,"endTime":1767684186,"taskType":4}]}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/download/item-1":
			time.Sleep(20 * time.Millisecond)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("bundle-content"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "bundle.zip")
	lookup := testLookupWithPortal(
		testPortalURL(server.URL, "org-tiup", "cluster-9", 1772776800, 1772777400),
		"CLINIC_OUTPUT_PATH", outputPath,
	)

	var progress bytes.Buffer
	var out bytes.Buffer
	err := runCollectedDataDownload(lookup, func() time.Time { return time.Unix(1772777400, 0) }, log.New(&progress, "", 0), &out)
	if err != nil {
		t.Fatalf("expected collected data download to succeed, got %v", err)
	}
	if !strings.Contains(progress.String(), "正在下载 bundle... 已耗时") {
		t.Fatalf("expected periodic download reminder, got=%q", progress.String())
	}
}

func TestCollectedDataDownloadEmitsPeriodicRebuildReminder(t *testing.T) {
	previousInterval := progressReminderInterval
	progressReminderInterval = 5 * time.Millisecond
	defer func() { progressReminderInterval = previousInterval }()

	var statusCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/clinic/api/v1/dashboard/clusters":
			w.Write([]byte(`{"items":[{"clusterID":"cluster-9","clusterName":"tiup-demo","orgID":"org-tiup","tenantID":"tenant-1","projectID":"","clusterProviderName":"","clusterRegionName":"","clusterDeployType":"tiup-cluster","clusterDeployTypeV2":"tiup-cluster","clusterStatus":"active","clusterCreatedAt":1,"capabilityHints":{}}]}`))
		case "/clinic/api/v1/orgs/org-tiup":
			w.Write([]byte(`{"id":"org-tiup","name":"tiup-org","type":""}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data":
			w.Write([]byte(`{"total":1,"dataInfos":[{"startTime":1767677814,"endTime":1767684186,"itemID":"item-1","filename":"bundle.zip","collectors":["log.std"],"haveLog":true,"haveMetric":true,"haveConfig":true}]}`))
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/data_status":
			switch statusCalls.Add(1) {
			case 1:
				w.Write([]byte(`{"items":[{"itemID":"item-1","status":0,"startTime":1767677814,"endTime":1767684186,"taskType":4}]}`))
			default:
				w.Write([]byte(`{"items":[{"itemID":"item-1","status":100,"startTime":1767677814,"endTime":1767684186,"taskType":4}]}`))
			}
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/rebuild":
			time.Sleep(20 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		case "/clinic/api/v1/orgs/org-tiup/clusters/cluster-9/download/item-1":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("bundle-content"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "bundle.zip")
	lookup := testLookupWithPortal(
		testPortalURL(server.URL, "org-tiup", "cluster-9", 1772776800, 1772777400),
		"CLINIC_OUTPUT_PATH", outputPath,
		"CLINIC_REBUILD_PROBE_INTERVAL", "20ms",
	)

	var progress bytes.Buffer
	var out bytes.Buffer
	err := runCollectedDataDownload(lookup, func() time.Time { return time.Unix(1772777400, 0) }, log.New(&progress, "", 0), &out)
	if err != nil {
		t.Fatalf("expected collected data download to succeed, got %v", err)
	}
	text := progress.String()
	if !strings.Contains(text, "正在等待 rebuild 完成...") {
		t.Fatalf("expected rebuild wait progress, got=%q", text)
	}
	if !strings.Contains(text, "正在等待 rebuild 完成... 已耗时") {
		t.Fatalf("expected periodic rebuild reminder, got=%q", text)
	}
}
