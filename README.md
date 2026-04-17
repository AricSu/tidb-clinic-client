# tidb-clinic-client

`tidb-clinic-client` is the V2 cluster-scoped Go SDK and CLI for TiDB Clinic.

V2 is a breaking simplification:

- the public SDK is cluster-scoped and capability-first
- `Client.Cloud`, `Client.DataProxy`, `Client.Loki`, and catalog-style public compatibility entry points are no longer part of the root client surface
- the public CLI only exposes capability-first command trees
- `tiup-cluster` collected-data behavior is inferred automatically from shared control-plane metadata

## Public SDK

The root client exposes one canonical public entry:

- `Client.Clusters`

Callers resolve once and then work from a bound `ClusterHandle`.

Platform rules:

- `Resolve(clusterID)` is the canonical path
- the SDK resolves through the shared `/dashboard/clusters2 -> /orgs/{id} -> /clusters/{id}` chain
- `org.type=="cloud"` stays on the cloud path
- `deployTypeV2/deployType=="tiup-cluster"` is treated as the tiup-cluster collected-data path

## Quick Start

```go
package main

import (
	"context"
	"log"
	"time"

	clinicapi "github.com/AricSu/tidb-clinic-client"
)

func main() {
	client, err := clinicapi.NewClientWithConfig(clinicapi.Config{
		BaseURL:     "https://clinic.pingcap.com",
		BearerToken: "token",
		Timeout:     10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	cluster, err := client.Clusters.Resolve(context.Background(), "cluster-9")
	if err != nil {
		log.Fatal(err)
	}

	result, err := cluster.Metrics.QueryRange(context.Background(), clinicapi.TimeSeriesQuery{
		Query: "sum(tidb_server_connections)",
		Start: 1772776800,
		End:   1772777400,
		Step:  "1m",
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = result
}
```

## Capability Surface

Target resolution:

- `Clusters.Resolve` is the canonical cluster-id-first resolve entry point
- it returns a `ClusterHandle`, which exposes direct cluster methods plus bound
capability clients. For lightweight observability, the handle also exposes
`Platform()`, `ClusterID()`, and `OrgID()`.

Cloud-only public capabilities:

- `Clusters.Search`
- `ClusterHandle.Detail`
- `ClusterHandle.Topology`
- `ClusterHandle.Events`
- `ClusterHandle.EventDetail`
- `ClusterHandle.SQLAnalytics.TopSQLSummary`
- `ClusterHandle.SQLAnalytics.TopSlowQueries`
- `ClusterHandle.SQLAnalytics.SlowQuerySamples`
- `ClusterHandle.SQLAnalytics.SlowQueryDetail`
- `ClusterHandle.SQLAnalytics.SQLStatements`
- `ClusterHandle.Profiling.ListGroups`
- `ClusterHandle.Profiling.Detail`
- `ClusterHandle.Profiling.ActionToken`
- `ClusterHandle.Profiling.Download`
- `ClusterHandle.Profiling.Fetch`
- `ClusterHandle.Diagnostics.ListPlanReplayer`
- `ClusterHandle.Diagnostics.ListOOMRecord`
- `ClusterHandle.Diagnostics.Download`

Cross-platform public capabilities:

- `ClusterHandle.Metrics.QueryRange`
- `ClusterHandle.Metrics.QueryInstant`
- `ClusterHandle.Metrics.QuerySeries`
- `ClusterHandle.Metrics.SeriesExists`
- `ClusterHandle.Logs.Query` / `QueryRange` / `Labels` / `LabelValues` on cloud
- `ClusterHandle.Logs.Search` on `tiup-cluster`
- `ClusterHandle.SQLAnalytics.Query` / `Schema` on cloud
- `ClusterHandle.SQLAnalytics.SlowQueryRecords` on `tiup-cluster`
- `ClusterHandle.Configs.Get` on `tiup-cluster`
- `ClusterHandle.Capabilities.Discover` on both paths

Capability discovery is the public contract for availability. Unsupported work returns `ErrUnsupported` from the client surface before transport errors leak through.

The stable capability set is:

- `cluster_detail`
- `topology`
- `events`
- `metrics`
- `logs`
- `sql_query`
- `schema`
- `topsql`
- `slow_query`
- `sql_statements`
- `configs`
- `profiling`
- `diagnostic_files`

## CLI

Install:

```bash
go install github.com/AricSu/tidb-clinic-client/cmd/clinic-client@latest
```

Top-level V2 commands:

```bash
clinic-client clusters ...
clinic-client metrics ...
clinic-client logs ...
clinic-client sql ...
clinic-client configs ...
clinic-client profiling ...
clinic-client diagnostics ...
clinic-client capabilities ...
```

Important subcommands:

```bash
clinic-client logs search
clinic-client sql slowquery-records
clinic-client configs get
clinic-client capabilities discover
```

Removed in V2:

- `clinic-client cloud ...`
- `clinic-client op ...`

## CLI Environment

Required for all runs:

- `CLINIC_API_KEY`
- `CLINIC_CLUSTER_ID`

Platform selection:

- the CLI always auto-resolves from `CLINIC_CLUSTER_ID`

Common query inputs:

- `CLINIC_METRICS_QUERY`
- `CLINIC_METRICS_MATCH`
- `CLINIC_RANGE_START`
- `CLINIC_RANGE_END`
- `CLINIC_RANGE_STEP`
- `CLINIC_QUERY_TIME`
- `CLINIC_TIMEOUT_SEC`

Shared SQL / logs / collected-data inputs:

- `CLINIC_LIMIT`
- `CLINIC_DESC`
- `CLINIC_LOG_PATTERN`
- `CLINIC_SLOWQUERY_ORDER_BY`

SQL analytics inputs:

- `CLINIC_COMPONENT`
- `CLINIC_INSTANCE`
- `CLINIC_TOPSQL_WINDOW`
- `CLINIC_TOPSQL_GROUP_BY`
- `CLINIC_SLOWQUERY_DIGEST`
- `CLINIC_SLOWQUERY_ID`
- `CLINIC_SLOWQUERY_CONNECTION_ID`
- `CLINIC_SLOWQUERY_TIMESTAMP`
- `CLINIC_SLOWQUERY_FIELDS`

Slow-query detail rules:

- on `tiup-cluster`, `sql slowquery-detail` accepts `CLINIC_SLOWQUERY_ID` and resolves the collected item automatically from `CLINIC_RANGE_START` and `CLINIC_RANGE_END`
- on cloud, `sql slowquery-detail` uses `CLINIC_SLOWQUERY_DIGEST + CLINIC_SLOWQUERY_CONNECTION_ID + CLINIC_SLOWQUERY_TIMESTAMP`

## Architecture

The public call model is:

- `Client/ClusterHandle -> resolve -> support check -> Clinic API client -> HTTP transport`

The SDK keeps the routing and retained-data decisions inside `internal/clinic`,
and keeps the remote API integration inside `internal/clinicapi`. The transport
boundary stays internal so request/response wiring can change without changing
the public SDK contract.

## Notes

- low-level transport is an internal implementation detail for V2 callers
- capability-native result types are the public contract
- tiup-cluster collected-data item selection for logs, slow-query records, and config snapshots happens automatically inside the SDK
- OP collected-data-backed public results expose retained-data provenance through
  `RetainedData`; legacy top-level compatibility fields are not part of the V2
  public contract
- deleted/tier-specific restrictions are surfaced through capability discovery and `ErrUnsupported`
