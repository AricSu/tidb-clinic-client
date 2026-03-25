package clinicapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	clinicapi "github.com/aric/tidb-clinic-client"
)

func ExampleNewClientWithConfig() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clinic/api/v1/orgs/org-1/clusters/cluster-9/data":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1,
				"dataInfos": []map[string]any{
					{
						"itemID":     "item-1",
						"filename":   "diag.tar.zst",
						"collectors": []string{"monitor.metric", "log.std"},
						"haveLog":    true,
						"haveMetric": true,
						"haveConfig": false,
						"startTime":  1772276800,
						"endTime":    1772277400,
					},
				},
			})
		case "/clinic/api/v1/data/metrics":
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
							},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := clinicapi.NewClientWithConfig(clinicapi.Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		panic(err)
	}

	items, err := client.Catalog.ListClusterData(context.Background(), clinicapi.ListClusterDataRequest{
		Context: clinicapi.RequestContext{
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
	})
	if err != nil {
		panic(err)
	}

	metrics, err := client.Metrics.QueryRange(context.Background(), clinicapi.MetricsQueryRangeRequest{
		Context: clinicapi.RequestContext{
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
		panic(err)
	}

	fmt.Printf("items=%d first_item=%s metric_series=%d first_value=%s\n",
		len(items),
		items[0].ItemID,
		len(metrics.Series),
		metrics.Series[0].Values[0].Value,
	)

	// Output:
	// items=1 first_item=item-1 metric_series=1 first_value=3034
}

func ExampleWithHooks() {
	var starts, dones, retries, failures int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total":     0,
			"dataInfos": []map[string]any{},
		})
	}))
	defer server.Close()

	client, err := clinicapi.NewClientWithConfig(clinicapi.Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		Timeout:     5 * time.Second,
		Hooks: clinicapi.Hooks{
			OnRequestStart: func(info clinicapi.RequestInfo) { starts++ },
			OnRequestDone:  func(result clinicapi.RequestResult) { dones++ },
			OnRetry:        func(retry clinicapi.RequestRetry) { retries++ },
			OnError:        func(failure clinicapi.RequestFailure) { failures++ },
		},
	})
	if err != nil {
		panic(err)
	}

	_, err = client.Catalog.ListClusterData(context.Background(), clinicapi.ListClusterDataRequest{
		Context: clinicapi.RequestContext{
			OrgID:     "org-1",
			ClusterID: "cluster-9",
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("starts=%d dones=%d retries=%d failures=%d\n", starts, dones, retries, failures)

	// Output:
	// starts=1 dones=1 retries=0 failures=0
}
