# Capability-First External SDK Design

Supersedes the public-abstraction direction of
[`2026-04-15-cloud-mcp-capability-expansion-design.md`](./2026-04-15-cloud-mcp-capability-expansion-design.md)
by keeping the Cloud retrieval coverage from that design while formalizing the
stable external SDK model.

## Summary

`tidb-clinic-client` should present itself as an external full-data SDK with:

- cluster-scoped capability-first public facades
- cluster-id-first target resolution
- compatibility low-level transport clients
- no analyzer or diagnosis logic

## Stable Public Model

Recommended entry points:

- `Client.Clusters`
- `Client.Clusters.Resolve(...)->ClusterHandle`
- `ClusterHandle.Metrics`
- `ClusterHandle.Logs`
- `ClusterHandle.SQLAnalytics`
- `ClusterHandle.Profiling`
- `ClusterHandle.Diagnostics`
- `ClusterHandle.Capabilities`

Compatibility layers that remain supported:

- `Client.Cloud`
- `Client.DataProxy`
- `Client.Loki`
- shared `RequestContext`-based APIs

## Canonical Targets

The public target model is:

- `ClusterIdentity`
- `ResolvedClusterTarget`
- `ControlPlaneTarget`
- `MetricsTarget`
- `LogsTarget`
- `SQLTarget`
- `ProfilingTarget`
- `DiagnosticsTarget`

The SDK owns the routing rules needed to derive those targets from a known
`cluster_id`.

## Capability Contract

Capability discovery returns a stable per-cluster descriptor with:

- `name`
- `available`
- `reason`
- `scope`
- `stability`
- `requires_parent_target`
- `requires_live_cluster`
- `tier_constraints`

The initial public capability set includes:

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
- `profiling`
- `diagnostic_files`

`sql_statements` is part of the stable public contract and is routed through
the same SQL analytics facade as the other SQL capabilities.

## Query Models

The stable capability-level query models are:

- `TimeSeriesQuery`
- `SQLQuery`
- `SchemaQuery`
- `SQLStatementsQuery`
- `QueryMetadata`
- `DownloadedArtifact`

The current backend still primarily lowers raw SQL. The public query models are
intentionally split by capability so future Data Service providers can adopt
them without reintroducing a generic catch-all query contract.

## CLI Shape

The preferred CLI tree is capability-first:

- `clinic-client clusters ...`
- `clinic-client metrics ...`
- `clinic-client logs ...`
- `clinic-client sql ...`
- `clinic-client profiling ...`
- `clinic-client diagnostics ...`
- `clinic-client capabilities ...`

Compatibility trees remain available:

- `clinic-client cloud ...`
- `clinic-client op ...`

## Non-Goals

- diagnosis or RCA generation
- workflow orchestration
- analyzer-specific result contracts
