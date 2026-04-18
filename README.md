# tidb-clinic-client

`tidb-clinic-client` is the Go SDK and CLI for the current TiDB Clinic command set.
The repository also vendors `compiler-rs` under [`compiler-rs/`](./compiler-rs) and
uses its embedded WASM build to power `metrics compile`.

## CLI

Build the local binary:

```bash
make
```

Or install from source:

```bash
go install github.com/AricSu/tidb-clinic-client/cmd/clinic-client@latest
```

Current command tree:

```bash
clinic-client cluster
clinic-client metrics query
clinic-client metrics compile
clinic-client slowquery
clinic-client op-pkgs list
clinic-client op-pkgs download
clinic-client cloud-events search
clinic-client cloud-logs search
clinic-client cloud-profilings list
clinic-client cloud-profilings download
clinic-client cloud-plan-replayers list
clinic-client cloud-plan-replayers download
clinic-client cloud-oom-records list
clinic-client cloud-oom-records download
```

Command intent:

- `cluster`: print cluster detail as raw JSON
- `metrics query`: run a metrics range query
- `metrics compile`: run the same metrics range query, then analyze the returned
  series with `compiler-rs` using automatic `line` / `group` selection
- `slowquery`: query slow queries for both cloud and OP clusters; cloud goes
  directly to the NGM slowquery API, while OP automatically selects a matching
  collected bundle
- `op-pkgs list` / `download`: list or download on-premise collected-data
  bundles
- `cloud-events search`: search cloud cluster events
- `cloud-logs search`: auto-dispatch between labels, label-values, and ranged
  log query
- `cloud-profilings list` / `download`: list or download cloud profiling
  artifacts
- `cloud-plan-replayers list` / `download`: list or download cloud plan
  replayer artifacts
- `cloud-oom-records list` / `download`: list or download cloud OOM record
  artifacts

## Environment

Base inputs used across commands:

- `CLINIC_PORTAL_URL`
- `CLINIC_API_KEY` for `clinic.pingcap.com`
- `CLINIC_CN_API_KEY` for `clinic.pingcap.com.cn`

Common command inputs:

- `metrics query` / `metrics compile`
  - `CLINIC_METRICS_QUERY`
  - `CLINIC_RANGE_STEP`
- `cloud-events search`
  - `CLINIC_EVENT_NAME`
  - `CLINIC_EVENT_SEVERITY`
  - also supports `--name` and `--severity`
- `slowquery`
  - `CLINIC_SLOWQUERY_ORDER_BY`
  - `CLINIC_SLOWQUERY_LIMIT`
  - `CLINIC_SLOWQUERY_DESC`
  - also supports `--order-by`, `--limit`, `--desc`
  - also accepts legacy `CLINIC_LIMIT` and `CLINIC_DESC`
- `cloud-logs search`
  - `CLINIC_LOKI_LABEL` to trigger label-values mode
  - `CLINIC_LOKI_QUERY`
  - `CLINIC_LOKI_LIMIT`
  - `CLINIC_LOKI_DIRECTION`
  - also supports `--label`, `--query`, `--limit`, `--direction`
- download commands
  - `CLINIC_OUTPUT_PATH`
  - `CLINIC_DIAGNOSTIC_KEY` for cloud diagnostic downloads
  - `CLINIC_PROFILE_TS`
  - `CLINIC_PROFILE_TYPE`
  - `CLINIC_PROFILE_COMPONENT`
  - `CLINIC_PROFILE_ADDRESS`
  - `CLINIC_PROFILE_DATA_FORMAT`

Notes:

- The CLI derives base URL and cluster ID from `CLINIC_PORTAL_URL`.
- When the portal URL includes `from` and `to`, the CLI reuses that time window
  automatically.
- `slowquery` skips item selection for cloud clusters and only auto-selects a
  collected bundle for non-cloud / OP deployments.
- `cloud-logs search` chooses mode in this order:
  `label-values -> query-range -> labels`.

## SDK

The public SDK is intentionally small and cluster-scoped.

Canonical flow:

1. Create a `Client`.
2. Resolve a cluster with `Client.Clusters.Resolve`.
3. Use the bound `ClusterHandle` capability clients.

Supported public APIs:

- `Client`, `Config`, auth / transport / retry options
- `Client.Clusters.Resolve`
- `ClusterHandle.Platform`, `ClusterHandle.ClusterID`, `ClusterHandle.OrgID`
- `ClusterHandle.Metrics.QueryRange`
- `ClusterHandle.SlowQueries.Query`
- `ClusterHandle.Logs.QueryRange`, `Labels`, `LabelValues`
- `ClusterHandle.CollectedData.List`, `Download`
- `ClusterHandle.Profiling.ListGroups`, `Fetch`
- `ClusterHandle.Diagnostics.ListPlanReplayer`, `ListOOMRecord`, `Download`

Example:

```go
package main

import (
	"context"
	"log"

	clinicapi "github.com/AricSu/tidb-clinic-client"
)

func main() {
	client, err := clinicapi.NewClientWithConfig(clinicapi.Config{
		BaseURL:     "https://clinic.pingcap.com",
		BearerToken: "token",
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

## compiler-rs

`compiler-rs` is the Rust time-series semantic kernel embedded by
`metrics compile`. It stays inside this repository as a workspace subtree rather
than a separate service.

Useful targets from the repository root:

- `make`: build `bin/clinic-client`
- `make test`: run Go tests and `compiler-rs` cargo tests
- `make wasm`: rebuild the embedded `compiler-rs` WASM asset
- `make viewer`: start the local regression viewer on port `8765`
- `make sync-cases`: regenerate compiler regression cases

If you want to work from inside `compiler-rs/`, the local `Makefile` forwards
`test`, `wasm`, `viewer`, and `sync-cases` back to the repo root.

## Development

This repository only keeps the SDK and CLI surface that backs the current
command tree. Do not reintroduce legacy command trees, compatibility shims, or
backend-shaped helper APIs that are no longer exposed.

Recommended checks:

```bash
make
make test
make vet
make check
```
