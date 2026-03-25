---
name: clinic-sdk-operation-router
description: Route tasks to the correct tidb-clinic-client method and respect cloud/op boundaries. Use when choosing an SDK method, CLI command, or request type.
---

## Purpose

Use this skill when a task asks you to fetch Clinic data through the SDK and
you need to decide which method to call.

This skill is for SDK navigation, not for diagnosis.

For the CLI distributed with this repository, treat `cloud` and `op` as the
only public root command groups.

## Core Rules

1. Start with the README support matrix and exported type names.
2. Assume the target cluster is already known.
3. Reject discovery asks such as `list_clusters`; they are out of scope for this SDK.
4. Prefer shared data-plane methods when the task can be expressed with `RequestContext`.
5. Use `Client.Cloud` only for cloud-only known-target control-plane or NGM requests.
6. Treat `Client.OP` as a placeholder namespace until real On-Premise-only SDK methods exist.
7. For CLI usage, prefer `clinic-client op ...` for On-Premise item-scoped workflows instead of exposing catalog directly.

## Routing Guide

### Shared Data Plane

Use these when the task applies to either Cloud or On-Premise hosted Clinic data:

- list uploaded data items:
  - `Client.Catalog.ListClusterData`
  - request: `ListClusterDataRequest`
- query metrics over a time range:
  - `Client.Metrics.QueryRange`
  - request: `MetricsQueryRangeRequest`
- query metrics with automatic split on max-samples responses:
  - `Client.Metrics.QueryRangeWithAutoSplit`
  - request: `MetricsQueryRangeRequest`
- retrieve slow queries for an uploaded item:
  - `Client.SlowQueries.Query`
  - request: `SlowQueryQueryRequest`
- search uploaded logs:
  - `Client.Logs.Search`
  - request: `LogSearchRequest`
- fetch uploaded config snapshots:
  - `Client.Configs.Get`
  - request: `ConfigGetRequest`

These methods use `RequestContext` with:

- `OrgType`
- `OrgID`
- `ClusterID`

Item-scoped operations additionally require `ItemID`.

### Cloud-Only

Use these only when the task is explicitly about cloud control-plane or NGM data:

- resolve cloud routing metadata from a known cluster id:
  - `Client.Cloud.GetCluster`
  - request: `CloudClusterLookupRequest`
- known-target cluster detail:
  - `Client.Cloud.GetClusterDetail`
  - request: `CloudClusterDetailRequest`
- known-target cloud events:
  - `Client.Cloud.QueryEvents`
  - request: `CloudEventQueryRequest`
- known-target single event detail:
  - `Client.Cloud.GetEventDetail`
  - request: `CloudEventDetailRequest`
- NGM TopSQL summary:
  - `Client.Cloud.GetTopSQLSummary`
  - request: `CloudTopSQLSummaryRequest`
- NGM top slow-query summary:
  - `Client.Cloud.GetTopSlowQueries`
  - request: `CloudTopSlowQueriesRequest`
- NGM slow-query list:
  - `Client.Cloud.ListSlowQueries`
  - request: `CloudSlowQueryListRequest`
- NGM single slow-query detail:
  - `Client.Cloud.GetSlowQueryDetail`
  - request: `CloudSlowQueryDetailRequest`

Cloud control-plane requests use `CloudTarget`.

Cloud NGM requests use `CloudNGMTarget`, which requires explicit routing values
such as provider, region, tenant, project, cluster, and deploy type.

If the caller only knows a cloud `cluster_id`, first call `Client.Cloud.GetCluster`
and then derive:

- `RequestContext()`
- `CloudTarget()`
- `CloudNGMTarget()`

## Selection Heuristics

- If the task says “artifact”, “uploaded item”, “itemID”, or refers to collectors,
  start in the shared data plane.
- If the task asks for raw metric series, use `QueryRange` or `QueryRangeWithAutoSplit`.
- If the task is about NGM TopSQL or cloud slow-query drill-down, use `Client.Cloud`.
- If the task is On-Premise-specific but still based on hosted Clinic data, continue using
  the shared data-plane methods, or use the `clinic-client op ...` workflows in the CLI.
- If the task asks for an On-Premise-only control-plane API, report that the SDK does not
  expose one yet.

### OP CLI Workflows

Use these when the task is specifically about the CLI rather than the Go SDK:

- search On-Premise (OP) logs with automatic item selection:
  - `clinic-client op logs`
- query On-Premise (OP) slow queries with automatic item selection:
  - `clinic-client op slowqueries`
- fetch On-Premise (OP) config snapshots with automatic item selection:
  - `clinic-client op configs`

These commands:

- require known `CLINIC_ORG_ID` and `CLINIC_CLUSTER_ID`
- treat `catalog` as an internal step
- do not expose `catalog` as a public command

### Cloud CLI Workflows

Use these when the task is specifically about the CLI and the target is
TiDB Cloud:

- metrics:
  - `clinic-client cloud metrics query-range`
- cluster detail:
  - `clinic-client cloud cluster-detail`
- events:
  - `clinic-client cloud events query`
  - `clinic-client cloud events detail`
- TopSQL:
  - `clinic-client cloud topsql summary`
- cloud slow queries:
  - `clinic-client cloud slowqueries top`
  - `clinic-client cloud slowqueries list`
  - `clinic-client cloud slowqueries detail`

## Anti-Patterns

- Do not invent discovery support.
- Do not guess missing cluster identity when the caller has not provided it.
- Do not treat `Client.OP` as if it already had implemented methods.
- Do not suggest removed root CLI paths such as `clinic-client metrics` or `clinic-client logs`.
- Do not expose `catalog` as a public CLI step for On-Premise flows.
- Do not route to `Client.Cloud` when the shared data plane already covers the task.
- Do not duplicate the SDK contract in prose when the exported request type already
  expresses the needed shape.
