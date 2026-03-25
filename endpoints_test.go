package clinicapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestCatalogListClusterDataSendsExpectedRequestAndDecodesResponse(t *testing.T) {
	var gotAuth, gotUA string
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotPath = r.URL.Path
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total": 1,
			"dataInfos": []map[string]any{
				{
					"startTime":  1772276800,
					"endTime":    1772277400,
					"itemID":     "item-1",
					"filename":   "diag.tar.zst",
					"collectors": []string{"monitor.metric", "log.std"},
					"haveLog":    true,
					"haveMetric": true,
					"haveConfig": false,
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		UserAgent:   "tidb-clinic-client-test",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	items, err := client.Catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err != nil {
		t.Fatalf("ListClusterData failed: %v", err)
	}
	if gotAuth != "Bearer token" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if gotUA != "tidb-clinic-client-test" {
		t.Fatalf("unexpected user-agent: %q", gotUA)
	}
	if gotPath != "/clinic/api/v1/orgs/org-1/clusters/cluster-9/data" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if len(items) != 1 || items[0].ItemID != "item-1" || !items[0].HaveMetric {
		t.Fatalf("unexpected decoded items: %+v", items)
	}
}

func TestCatalogListClusterDataUsesSDKDefaultUserAgent(t *testing.T) {
	var gotUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total":     0,
			"dataInfos": []map[string]any{},
		})
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
	_, err = client.Catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err != nil {
		t.Fatalf("ListClusterData failed: %v", err)
	}
	if gotUA != "tidb-clinic-client" {
		t.Fatalf("unexpected default user-agent: %q", gotUA)
	}
}

func TestCloudGetClusterDetailSendsExpectedRequestAndDecodesResponse(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "cluster-9",
			"name": "production",
			"components": map[string]any{
				"COMPONENT_TYPE_TIDB": map[string]any{
					"replicas":            2,
					"tierName":            "standard",
					"storageInstanceType": "",
				},
				"COMPONENT_TYPE_TIKV": map[string]any{
					"replicas":            3,
					"tierName":            "performance",
					"storageInstanceType": "gp3",
					"storages": map[string]any{
						"data": map[string]any{
							"iops": 3000,
						},
					},
				},
			},
		})
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
	result, err := client.Cloud.GetClusterDetail(context.Background(), CloudClusterDetailRequest{
		Target: CloudTarget{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err != nil {
		t.Fatalf("GetClusterDetail failed: %v", err)
	}
	if gotPath != "/clinic/api/v1/orgs/org-1/clusters/cluster-9" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if result.ID != "cluster-9" || result.Name != "production" {
		t.Fatalf("unexpected cluster detail: %+v", result)
	}
	if result.Components[CloudClusterComponentTypeTiKV].StorageIOPS != 3000 {
		t.Fatalf("expected tikv iops to be decoded, got=%+v", result.Components)
	}
}

func TestCloudQueryEventsSendsExpectedRequestAndDecodesResponse(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total": 1,
			"activities": []map[string]any{
				{
					"event_id":         "evt-1",
					"name":             "modify_cluster_size",
					"display_name":     "Modify Cluster Size",
					"calibration_time": 1772776800,
					"payload": map[string]any{
						"detail": map[string]any{
							"status": "Start",
						},
					},
				},
			},
		})
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
	result, err := client.Cloud.QueryEvents(context.Background(), CloudEventsRequest{
		Target:    CloudTarget{OrgID: "org-1", ClusterID: "cluster-9"},
		StartTime: 1772776800,
		EndTime:   1772777400,
	})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if gotPath != "/clinic/api/v1/activityhub/applications/org-1/targets/cluster-9/activities" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotQuery.Get("begin_ts") != "1772776800" || gotQuery.Get("end_ts") != "1772777400" {
		t.Fatalf("unexpected query params: %+v", gotQuery)
	}
	if result.Total != 1 || len(result.Events) != 1 || result.Events[0].EventID != "evt-1" {
		t.Fatalf("unexpected event result: %+v", result)
	}
}

func TestCloudGetEventDetailExtractsPayloadDetail(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{
			"payload": map[string]any{
				"detail": map[string]any{
					"status": "Start",
					"reason": "manual",
				},
			},
		})
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
	result, err := client.Cloud.GetEventDetail(context.Background(), CloudEventDetailRequest{
		Target:  CloudTarget{OrgID: "org-1", ClusterID: "cluster-9"},
		EventID: "evt-1",
	})
	if err != nil {
		t.Fatalf("GetEventDetail failed: %v", err)
	}
	if gotPath != "/clinic/api/v1/activityhub/applications/org-1/targets/cluster-9/activities/evt-1" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if result["status"] != "Start" || result["reason"] != "manual" {
		t.Fatalf("unexpected event detail: %+v", result)
	}
}

func TestCloudGetClusterSendsExpectedRequestAndDecodesResponse(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"clusterID":           "cluster-9",
					"clusterName":         "production",
					"clusterProviderName": "aws",
					"clusterRegionName":   "us-east-1",
					"clusterDeployType":   "dedicated",
					"orgID":               "org-1",
					"tenantID":            "tenant-1",
					"projectID":           "project-1",
					"clusterCreatedAt":    1772776000,
					"clusterDeletedAt":    nil,
					"clusterStatus":       "available",
				},
			},
		})
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
	result, err := client.Cloud.GetCluster(context.Background(), CloudClusterLookupRequest{
		ClusterID: "cluster-9",
	})
	if err != nil {
		t.Fatalf("GetCluster failed: %v", err)
	}
	if gotPath != "/clinic/api/v1/dashboard/clusters2" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotQuery.Get("cluster_id") != "cluster-9" || gotQuery.Get("limit") != "1" || gotQuery.Get("page") != "1" {
		t.Fatalf("unexpected cluster lookup query params: %+v", gotQuery)
	}
	if result.ClusterID != "cluster-9" || result.OrgID != "org-1" || result.Provider != "aws" || result.Region != "us-east-1" {
		t.Fatalf("unexpected cloud cluster result: %+v", result)
	}
}

func TestCloudClusterConversionHelpers(t *testing.T) {
	cluster := CloudCluster{
		ClusterID:  "cluster-9",
		OrgID:      "org-1",
		TenantID:   "tenant-1",
		ProjectID:  "project-1",
		Provider:   "aws",
		Region:     "us-east-1",
		DeployType: "dedicated",
	}

	reqCtx := cluster.RequestContext()
	if reqCtx.OrgType != "cloud" || reqCtx.OrgID != "org-1" || reqCtx.ClusterID != "cluster-9" {
		t.Fatalf("unexpected request context: %+v", reqCtx)
	}
	target := cluster.CloudTarget()
	if target.OrgID != "org-1" || target.ClusterID != "cluster-9" {
		t.Fatalf("unexpected cloud target: %+v", target)
	}
	ngmTarget := cluster.CloudNGMTarget()
	if ngmTarget.TenantID != "tenant-1" || ngmTarget.ProjectID != "project-1" || ngmTarget.Provider != "aws" || ngmTarget.Region != "us-east-1" || ngmTarget.DeployType != "dedicated" {
		t.Fatalf("unexpected cloud ngm target: %+v", ngmTarget)
	}
}

func TestCloudGetTopSQLSummarySendsExpectedNGMRequestAndDecodesResponse(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	var gotHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		gotHeaders = r.Header.Clone()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"sql_digest":           "sql-1",
					"sql_text":             "select * from t",
					"cpu_time_ms":          123.4,
					"exec_count_per_sec":   2.5,
					"duration_per_exec_ms": 10.2,
					"scan_records_per_sec": 100.1,
					"scan_indexes_per_sec": 10.0,
					"plans": []map[string]any{
						{
							"plan_digest":          "plan-1",
							"plan_text":            "IndexLookUp",
							"timestamp_sec":        []int64{1772776800},
							"cpu_time_ms":          []int64{88},
							"exec_count_per_sec":   1.1,
							"duration_per_exec_ms": 9.5,
							"scan_records_per_sec": 50.0,
							"scan_indexes_per_sec": 5.0,
						},
					},
				},
			},
		})
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
	result, err := client.Cloud.GetTopSQLSummary(context.Background(), CloudTopSQLSummaryRequest{
		Target: CloudNGMTarget{
			Provider:   "aws",
			Region:     "us-east-1",
			TenantID:   "tenant-1",
			ProjectID:  "project-1",
			ClusterID:  "cluster-9",
			DeployType: "dedicated",
		},
		Component: "tikv",
		Instance:  "tikv-0",
		Start:     "1772776800",
		End:       "1772777400",
	})
	if err != nil {
		t.Fatalf("GetTopSQLSummary failed: %v", err)
	}
	if gotPath != "/ngm/api/v1/topsql/summary" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotHeaders.Get("X-Provider") != "aws" ||
		gotHeaders.Get("X-Region") != "us-east-1" ||
		gotHeaders.Get("X-Org-Id") != "tenant-1" ||
		gotHeaders.Get("X-Project-Id") != "project-1" ||
		gotHeaders.Get("X-Cluster-Id") != "cluster-9" ||
		gotHeaders.Get("X-Deploy-Type") != "dedicated" {
		t.Fatalf("unexpected ngm headers: %+v", gotHeaders)
	}
	if gotQuery.Get("instance_type") != "tikv" || gotQuery.Get("top") != "5" || gotQuery.Get("window") != "60s" || gotQuery.Get("group_by") != "query" {
		t.Fatalf("unexpected topsql query params: %+v", gotQuery)
	}
	if len(result) != 1 || result[0].SQLDigest != "sql-1" || len(result[0].Plans) != 1 {
		t.Fatalf("unexpected topsql result: %+v", result)
	}
}

func TestCloudGetTopSlowQueriesSendsExpectedNGMRequestAndDecodesResponse(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"db":             "test",
				"sql_digest":     "sql-1",
				"sql_text":       "select * from t",
				"statement_type": "Select",
				"count":          3,
				"sum_latency":    1.5,
				"max_latency":    0.8,
				"avg_latency":    0.5,
				"sum_memory":     128,
				"max_memory":     64,
				"avg_memory":     42,
				"sum_disk":       1000,
				"max_disk":       500,
				"avg_disk":       333,
			},
		})
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
	result, err := client.Cloud.GetTopSlowQueries(context.Background(), CloudTopSlowQueriesRequest{
		Target: CloudNGMTarget{
			Provider:   "aws",
			Region:     "us-east-1",
			TenantID:   "tenant-1",
			ProjectID:  "project-1",
			ClusterID:  "cluster-9",
			DeployType: "dedicated",
		},
		Start: "1772776800",
	})
	if err != nil {
		t.Fatalf("GetTopSlowQueries failed: %v", err)
	}
	if gotPath != "/ngm/api/v1/slow_query/stats" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotQuery.Get("hours") != "1" || gotQuery.Get("order_by") != "sum_latency" || gotQuery.Get("limit") != "10" {
		t.Fatalf("unexpected top slow query params: %+v", gotQuery)
	}
	if len(result) != 1 || result[0].SQLDigest != "sql-1" || result[0].MaxLatency != 0.8 {
		t.Fatalf("unexpected top slow query result: %+v", result)
	}
}

func TestCloudListSlowQueriesSendsExpectedNGMRequestAndDecodesResponse(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"digest":        "sql-1",
				"query":         "select * from t",
				"timestamp":     "1772777000",
				"query_time":    1.23,
				"memory_max":    64,
				"request_count": 10,
				"connection_id": "conn-1",
			},
		})
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
	result, err := client.Cloud.ListSlowQueries(context.Background(), CloudSlowQueryListRequest{
		Target: CloudNGMTarget{
			Provider:   "aws",
			Region:     "us-east-1",
			TenantID:   "tenant-1",
			ProjectID:  "project-1",
			ClusterID:  "cluster-9",
			DeployType: "dedicated",
		},
		Digest:  "sql-1",
		Start:   "1772776800",
		End:     "1772777400",
		OrderBy: "query_time",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("ListSlowQueries failed: %v", err)
	}
	if gotPath != "/ngm/api/v1/slow_query/list" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotQuery.Get("digest") != "sql-1" || gotQuery.Get("fields") == "" || gotQuery.Get("desc") != "true" {
		t.Fatalf("unexpected slow query list params: %+v", gotQuery)
	}
	if len(result) != 1 || result[0].ConnectionID != "conn-1" || result[0].Query != "select * from t" {
		t.Fatalf("unexpected slow query list result: %+v", result)
	}
}

func TestCloudGetSlowQueryDetailSendsExpectedNGMRequestAndDecodesResponse(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"digest":       "sql-1",
			"query":        "select * from t",
			"plan":         "IndexLookUp",
			"wait_time":    0.2,
			"process_time": 0.8,
		})
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
	result, err := client.Cloud.GetSlowQueryDetail(context.Background(), CloudSlowQueryDetailRequest{
		Target: CloudNGMTarget{
			Provider:   "aws",
			Region:     "us-east-1",
			TenantID:   "tenant-1",
			ProjectID:  "project-1",
			ClusterID:  "cluster-9",
			DeployType: "dedicated",
		},
		Digest:       "sql-1",
		ConnectionID: "conn-1",
		Timestamp:    "1772777000",
	})
	if err != nil {
		t.Fatalf("GetSlowQueryDetail failed: %v", err)
	}
	if gotPath != "/ngm/api/v1/slow_query/detail" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotQuery.Get("digest") != "sql-1" || gotQuery.Get("connect_id") != "conn-1" || gotQuery.Get("timestamp") != "1772777000" {
		t.Fatalf("unexpected slow query detail params: %+v", gotQuery)
	}
	if result["plan"] != "IndexLookUp" || result["wait_time"] != 0.2 {
		t.Fatalf("unexpected slow query detail result: %+v", result)
	}
}

func TestMetricsQueryRangeSendsHeadersAndQueryAndDecodesMatrixResponse(t *testing.T) {
	var gotHeaders http.Header
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		gotQuery = r.URL.Query()
		if r.URL.Path != "/clinic/api/v1/data/metrics" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    "success",
			"isPartial": false,
			"data": map[string]any{
				"resultType": "matrix",
				"result": []map[string]any{
					{
						"metric": map[string]string{"instance": "tidb-0"},
						"values": [][]any{
							{float64(1772776843), "3034"},
							{float64(1772776903), "2518"},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		UserAgent:   "tidb-clinic-client-test",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	result, err := client.Metrics.QueryRange(context.Background(), MetricsQueryRangeRequest{
		Context: RequestContext{
			OrgType:   "cloud",
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
		Query: "sum(tidb_server_connections)",
		Start: 1772776800,
		End:   1772777400,
		Step:  "1m",
	})
	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}
	if gotHeaders.Get("Authorization") != "Bearer token" {
		t.Fatalf("unexpected auth header: %q", gotHeaders.Get("Authorization"))
	}
	if gotHeaders.Get("X-OrgType") != "cloud" || gotHeaders.Get("X-OrgID") != "org-1" || gotHeaders.Get("X-ClusterID") != "cluster-9" {
		t.Fatalf("unexpected routing headers: %+v", gotHeaders)
	}
	if gotQuery.Get("query") != "sum(tidb_server_connections)" || gotQuery.Get("start") != "1772776800" || gotQuery.Get("end") != "1772777400" || gotQuery.Get("step") != "1m" {
		t.Fatalf("unexpected query params: %+v", gotQuery)
	}
	if result.ResultType != "matrix" || len(result.Series) != 1 || len(result.Series[0].Values) != 2 {
		t.Fatalf("unexpected decoded result: %+v", result)
	}
}

func TestMetricsQueryRangeWithAutoSplitSplitsAndMergesResults(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path != "/clinic/api/v1/data/metrics" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		query := r.URL.Query()
		if callCount == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": "cannot select more than -search.maxSamplesPerQuery",
			})
			return
		}
		switch {
		case query.Get("start") == "1772776800" && query.Get("end") == "1772777100":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":    "success",
				"isPartial": false,
				"data": map[string]any{
					"resultType": "matrix",
					"result": []map[string]any{
						{
							"metric": map[string]string{"instance": "tidb-0"},
							"values": [][]any{
								{float64(1772776800), "10"},
							},
						},
					},
				},
			})
		case query.Get("start") == "1772777100" && query.Get("end") == "1772777400":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":    "success",
				"isPartial": false,
				"data": map[string]any{
					"resultType": "matrix",
					"result": []map[string]any{
						{
							"metric": map[string]string{"instance": "tidb-0"},
							"values": [][]any{
								{float64(1772777400), "20"},
							},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected split range: %+v", query)
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
	result, err := client.Metrics.QueryRangeWithAutoSplit(context.Background(), MetricsQueryRangeRequest{
		Context: RequestContext{
			OrgType:   "cloud",
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
		Query: "sum(tidb_server_connections)",
		Start: 1772776800,
		End:   1772777400,
		Step:  "1m",
	})
	if err != nil {
		t.Fatalf("QueryRangeWithAutoSplit failed: %v", err)
	}
	if callCount != 3 {
		t.Fatalf("expected 3 metric requests, got=%d", callCount)
	}
	if result.ResultType != "matrix" || len(result.Series) != 1 || len(result.Series[0].Values) != 2 {
		t.Fatalf("unexpected merged metrics result: %+v", result)
	}
	if result.Series[0].Values[0].Timestamp != 1772776800 || result.Series[0].Values[1].Timestamp != 1772777400 {
		t.Fatalf("unexpected merged metric samples: %+v", result.Series[0].Values)
	}
}

func TestSlowQueriesQuerySendsExpectedRequestAndDecodesResponse(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	var gotHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		gotHeaders = r.Header.Clone()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total": 1,
			"slowQueries": []map[string]any{
				{
					"digest":     "digest-1",
					"query":      "select * from t where id = ?",
					"queryTime":  1.77,
					"execCount":  3,
					"user":       "root",
					"db":         "test",
					"tableNames": []string{"test.t"},
					"indexNames": []string{"idx_id"},
					"sourceRef":  "item-1",
				},
			},
		})
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
	result, err := client.SlowQueries.Query(context.Background(), SlowQueryRequest{
		Context: RequestContext{
			OrgType:   "op",
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
		ItemID:    "item-1",
		StartTime: 1772776800,
		EndTime:   1772777400,
		OrderBy:   "time",
		Desc:      true,
		Limit:     20,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if gotPath != "/clinic/api/v1/data/slowqueries" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotHeaders.Get("X-OrgType") != "op" || gotHeaders.Get("X-OrgID") != "org-1" || gotHeaders.Get("X-ClusterID") != "cluster-9" {
		t.Fatalf("unexpected routing headers: %+v", gotHeaders)
	}
	if gotQuery.Get("itemID") != "item-1" || gotQuery.Get("startTime") != "1772776800" || gotQuery.Get("endTime") != "1772777400" || gotQuery.Get("orderBy") != "time" || gotQuery.Get("desc") != "true" || gotQuery.Get("limit") != "20" {
		t.Fatalf("unexpected query params: %+v", gotQuery)
	}
	if result.Total != 1 || len(result.Records) != 1 || result.Records[0].Digest != "digest-1" {
		t.Fatalf("unexpected decoded slowquery result: %+v", result)
	}
}

func TestLogsSearchSendsExpectedRequestAndDecodesResponse(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total": 1,
			"logs": []map[string]any{
				{
					"timestamp": 1772776843,
					"component": "tidb",
					"level":     "ERROR",
					"message":   "region unavailable",
					"sourceRef": "item-1",
				},
			},
		})
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
	result, err := client.Logs.Search(context.Background(), LogSearchRequest{
		Context: RequestContext{
			OrgType:   "op",
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
		ItemID:    "item-1",
		StartTime: 1772776800,
		EndTime:   1772777400,
		Pattern:   "region unavailable",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if gotPath != "/clinic/api/v1/data/logs" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotQuery.Get("itemID") != "item-1" || gotQuery.Get("pattern") != "region unavailable" || gotQuery.Get("limit") != "10" {
		t.Fatalf("unexpected query params: %+v", gotQuery)
	}
	if result.Total != 1 || len(result.Records) != 1 || result.Records[0].Component != "tidb" {
		t.Fatalf("unexpected decoded logs result: %+v", result)
	}
}

func TestConfigsGetSendsExpectedRequestAndDecodesResponse(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"configs": []map[string]any{
				{
					"component": "tidb",
					"key":       "token-limit",
					"value":     "1000",
					"sourceRef": "item-1",
				},
			},
		})
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
	result, err := client.Configs.Get(context.Background(), ConfigRequest{
		Context: RequestContext{
			OrgType:   "op",
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
		ItemID: "item-1",
	})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if gotPath != "/clinic/api/v1/data/config" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotQuery.Get("itemID") != "item-1" {
		t.Fatalf("unexpected query params: %+v", gotQuery)
	}
	if len(result.Entries) != 1 || result.Entries[0].Key != "token-limit" {
		t.Fatalf("unexpected config snapshot: %+v", result)
	}
}
