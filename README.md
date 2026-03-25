# tidb-clinic-client

`tidb-clinic-client` is a small Go SDK for the TiDB Clinic API.

It is intentionally narrow:

- transport and typed endpoint access only
- no workflow orchestration
- no diagnosis logic
- no application-specific integration logic

## Status

This repository is shaped for an initial public SDK release.

Current scope:

- cloud cluster lookup by known `cluster_id`
- hosted data catalog lookup
- metrics range queries
- metrics range queries with automatic time splitting on max-samples responses
- slow query retrieval
- log retrieval
- config retrieval
- known-target TiDB Cloud cluster detail lookup
- known-target TiDB Cloud event query and event detail lookup
- known-target TiDB Cloud NGM TopSQL summary
- known-target TiDB Cloud NGM top slow-query summary
- known-target TiDB Cloud NGM slow-query list and detail
- CLI workflows for On-Premise (OP) environments that resolve item IDs from catalog automatically
- reserved `Client.OP` placeholder for future OP-only APIs
- authentication providers
- error classification
- request lifecycle hooks

## Installation

```bash
go get github.com/AricSu/tidb-clinic-client
```

## CLI Installation

```bash
go install github.com/AricSu/tidb-clinic-client/cmd/clinic-client@latest
```

## CLI Usage

The public CLI exposes these command paths:

```bash
clinic-client cloud metrics query-range
clinic-client op metrics query-range
clinic-client op logs
clinic-client op slowqueries
clinic-client op configs
clinic-client cloud cluster-detail
clinic-client cloud events query
clinic-client cloud events detail
clinic-client cloud topsql summary
clinic-client cloud slowqueries top
clinic-client cloud slowqueries list
clinic-client cloud slowqueries detail
```

It uses the same `CLINIC_*` environment variables documented below.
For TiDB Cloud commands, the CLI may resolve routing metadata from
`CLINIC_CLUSTER_ID` via `Client.Cloud.GetCluster`, but `get-cluster` itself is
not exposed as a public CLI command.
For On-Premise (OP) commands, the CLI may resolve an item ID from
`Client.Catalog.ListClusterData`, but `catalog` itself is not exposed as a
public CLI command.
If `CLINIC_API_BASE_URL` is omitted, the CLI defaults to
`https://clinic.pingcap.com`.

The root CLI is intentionally platform-first:

- `clinic-client cloud ...`
- `clinic-client op ...`

## Make Targets

```bash
make
make all
make fmt
make test
make vet
make build-cli
make run-cli
```

- `make` prints the available targets
- `make all` runs the full local verification/build path

## Import

The package name is `clinicapi`, so consumers typically write:

```go
import clinicapi "github.com/AricSu/tidb-clinic-client"
```

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

	result, err := client.Metrics.QueryRange(context.Background(), clinicapi.MetricsQueryRangeRequest{
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
		log.Fatal(err)
	}

	_ = result
}
```

See `example_test.go` for runnable examples.

## Error Model

The client returns typed SDK errors with:

- `ErrorClass`
- retryability
- HTTP status code when available
- endpoint context

Helpers:

```go
clinicapi.IsRetryable(err)
clinicapi.ClassOf(err)
```

## Observability

The client supports:

- standard-library logger injection
- request lifecycle hooks

Hooks let callers attach metrics or tracing without hard-coupling this SDK to a specific telemetry stack.

## Authentication and User-Agent

- `AuthProvider` is the normalized runtime auth model
- `BearerToken` is supported as a convenience construction path
- the default `User-Agent` is `tidb-clinic-client`

This keeps the common token-based setup short while making the constructed
client state match the real transport model.

## Platform Scope

Current support matrix:

| Surface | Method / Entry Point | Cloud | OP | Notes |
|---|---|---:|---:|---|
| Shared data plane | `Client.Catalog.ListClusterData` | Yes | Yes | Hosted data catalog for a known target |
| Shared data plane | `Client.Metrics.QueryRange` | Yes | Yes | Routed by `RequestContext` |
| Shared data plane | `Client.Metrics.QueryRangeWithAutoSplit` | Yes | Yes | Same as above, with max-samples splitting |
| Shared data plane | `Client.SlowQueries.Query` | Yes | Yes | Uploaded-item slow query retrieval |
| Shared data plane | `Client.Logs.Search` | Yes | Yes | Uploaded-item log retrieval |
| Shared data plane | `Client.Configs.Get` | Yes | Yes | Uploaded-item config retrieval |
| Cloud-only | `Client.Cloud.GetCluster` | Yes | No | Resolve cloud routing metadata from a known `cluster_id` |
| Cloud-only | `Client.Cloud.GetClusterDetail` | Yes | No | Known-target cloud control-plane lookup |
| Cloud-only | `Client.Cloud.QueryEvents` | Yes | No | Known-target cloud activity query |
| Cloud-only | `Client.Cloud.GetEventDetail` | Yes | No | Known-target cloud event detail |
| Cloud-only | `Client.Cloud.GetTopSQLSummary` | Yes | No | NGM TopSQL summary |
| Cloud-only | `Client.Cloud.GetTopSlowQueries` | Yes | No | NGM slow query aggregate summary |
| Cloud-only | `Client.Cloud.ListSlowQueries` | Yes | No | NGM digest-scoped slow query list |
| Cloud-only | `Client.Cloud.GetSlowQueryDetail` | Yes | No | NGM single slow query detail |
| Reserved | `Client.OP` | No methods yet | No methods yet | Namespace reserved for future OP-only APIs |

CLI workflow support:

| Surface | Command | Cloud | OP | Notes |
|---|---|---:|---:|---|
| OP workflow | `clinic-client op logs` | No | Yes | Resolves an item ID from catalog automatically |
| OP workflow | `clinic-client op slowqueries` | No | Yes | Resolves an item ID automatically and prefers `log.slow` |
| OP workflow | `clinic-client op configs` | No | Yes | Resolves an item ID automatically and prefers the latest config snapshot |

Interpretation:

- Current On-Premise support lives in the shared `RequestContext`-based APIs.
- `Client.Cloud` is the place for cloud-only platform APIs.
- `Client.OP` exists only as a future On-Premise-only SDK extension point today.
- The current OP CLI commands are workflow wrappers over shared SDK APIs, not `Client.OP` methods.

Still intentionally out of scope:

- cluster discovery such as `list_clusters`
- workflow diagnosis logic
- automatic NGM target discovery or routing derivation

## Agent Guidance

If an agent needs to choose the right SDK entry point, the preferred sources are:

- this README support matrix
- `skills/clinic-sdk-operation-router/SKILL.md`
- exported Go request and response type names

## Local CLI Testing

For local CLI testing:

1. copy `.env.example` to `.env`
2. fill in your real Clinic values
3. export the variables into your shell
4. run the CLI

Example:

```bash
cp .env.example .env
$EDITOR .env
set -a
source .env
set +a
go run ./cmd/clinic-client cloud metrics query-range
```

Notes:

- `.env` is ignored by Git and should stay local
- `.env.example` is the committed template
- `CLINIC_API_BASE_URL` is optional; the CLI defaults to `https://clinic.pingcap.com`
- the CLI currently exercises `Client.Metrics.QueryRangeWithAutoSplit`
- for TiDB Cloud runs, `CLINIC_ORG_TYPE` defaults to `cloud`
- for TiDB Cloud runs, `CLINIC_ORG_ID` may be omitted; the CLI resolves it from `CLINIC_CLUSTER_ID` via `Client.Cloud.GetCluster`
- for On-Premise workflow commands, `CLINIC_ORG_TYPE=op`, `CLINIC_ORG_ID`, and `CLINIC_CLUSTER_ID` are required, but `CLINIC_ITEM_ID` is not
- if `CLINIC_RANGE_START` and `CLINIC_RANGE_END` are omitted, it uses the last 10 minutes by default
- `make run-cli` is the local shortcut for `clinic-client cloud metrics query-range`

Additional environment variables for platform-specific commands:

```bash
# On-Premise (OP) workflow commands
# These commands do not require CLINIC_ITEM_ID.
CLINIC_ORG_TYPE=op
CLINIC_ORG_ID=replace-me
CLINIC_CLUSTER_ID=replace-me
CLINIC_LIMIT=10
CLINIC_DESC=true
CLINIC_SLOWQUERY_ORDER_BY=queryTime
CLINIC_LOG_PATTERN=error

# Cloud events detail
CLINIC_EVENT_ID=replace-me

# Cloud TopSQL
CLINIC_CLOUD_COMPONENT=tidb
CLINIC_CLOUD_INSTANCE=tidb-0
CLINIC_CLOUD_START=1772776800
CLINIC_CLOUD_END=1772777400
CLINIC_CLOUD_TOP=5
CLINIC_CLOUD_WINDOW=60s
CLINIC_CLOUD_GROUP_BY=query

# Cloud slow query commands
CLINIC_CLOUD_HOURS=1
CLINIC_CLOUD_ORDER_BY=sum_latency
CLINIC_CLOUD_LIMIT=10
CLINIC_CLOUD_DIGEST=replace-me
CLINIC_CLOUD_DESC=true
CLINIC_CLOUD_FIELDS=query,timestamp,query_time,memory_max,request_count,connection_id
CLINIC_CLOUD_CONNECTION_ID=replace-me
CLINIC_CLOUD_TIMESTAMP=replace-me
```

## Local Module Development

When developing another Go module against a local checkout of this repository:

```go
require github.com/AricSu/tidb-clinic-client v0.0.0
replace github.com/AricSu/tidb-clinic-client => ../tidb-clinic-client
```

## Versioning

This repository is intended to have its own tags and changelog.

Until the first tagged release, the API should be treated as pre-1.0.

