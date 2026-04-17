# Cloud MCP Capability Expansion Design

## Summary

Expand `tidb-clinic-client` from a metrics-first Clinic SDK into a complete Cloud retrieval substrate aligned with the broader Clinic API capability model.

This design keeps the repository below the analyzer layer:

- typed SDK access
- CLI workflows
- deterministic routing helpers
- no diagnosis logic
- no `ClinicAnalysisResult`

## Goals

- Cover the Cloud-facing retrieval surfaces exposed by `nutshell` `clinic-api`
- Keep Cloud routing logic deterministic and reusable
- Reuse one transport, error, and request tracing model across all surfaces
- Expose every major Cloud retrieval capability through both SDK and CLI

## Non-Goals

- No diagnosis text or analyzer contracts
- No workflow orchestration beyond deterministic helper composition
- No attempt to merge broader analyzer or workflow assets into this first pass

## Capability Additions

### Cloud metadata and routing

- Enrich `CloudCluster` with:
  - `ParentID`
  - `DeployTypeV2`
  - provider / region / org / tenant / project identifiers needed by NGM and control-plane routes
- Add Cloud cluster search/list support
- Add resource-pool component lookup
- Add topology helper behavior for:
  - dedicated
  - premium / byoc
  - resource-pool
  - shared / starter / essential rejection

### Cloud retrieval surfaces

- Add `Client.DataProxy`
  - `Query`
  - `Schema`
- Add `Client.Loki`
  - `Query`
  - `QueryRange`
  - `Labels`
  - `LabelValues`
- Extend `Client.Cloud`
  - `SearchClusters`
  - `GetResourcePoolComponents`
  - `GetTopology`
  - `ListProfileGroups`
  - `GetProfileGroupDetail`
  - `GetProfileActionToken`
  - `DownloadProfile`
  - `FetchProfile`
  - `ListPlanReplayerArtifacts`
  - `ListOOMRecordArtifacts`
  - `DownloadDiagnosticArtifact`

### Helper layer

- Add helper methods that encode stable Cloud rules:
  - resolve exact cluster metadata by cluster ID
  - derive control-plane targets
  - derive NGM targets
  - derive Premium / BYOC resource-pool targets
  - metrics parent-cluster fallback query shaping
  - metrics series discovery helper
  - default output-path helpers for profile and diagnostic downloads

## Public API Shape

### Root client

- Add `Client.DataProxy`
- Add `Client.Loki`

### Data Proxy

- `DataProxyQueryRequest`
- `DataProxyQueryResult`
- `DataProxySchemaRequest`
- `DataProxySchemaResult`

### Loki

- `LokiQueryRequest`
- `LokiQueryResult`
- `LokiQueryRangeRequest`
- `LokiQueryRangeResult`
- `LokiLabelsRequest`
- `LokiLabelsResult`
- `LokiLabelValuesRequest`
- `LokiLabelValuesResult`

### Cloud profiling

- `CloudProfileGroupsRequest`
- `CloudProfileGroupsResult`
- `CloudProfileGroup`
- `CloudProfileGroupDetailRequest`
- `CloudProfileGroupDetail`
- `CloudProfileTargetProfile`
- `CloudProfileActionTokenRequest`
- `CloudProfileDownloadRequest`
- `CloudProfileFetchRequest`
- `CloudDownloadedArtifact`

### Cloud diagnostic files

- `CloudDiagnosticListRequest`
- `CloudDiagnosticRecordGroup`
- `CloudDiagnosticFile`
- `CloudDiagnosticListResult`
- `CloudDiagnosticDownloadRequest`

## Routing Rules

### Control plane

- `/clinic/api/v1/orgs/{orgID}/...` uses internal `OrgID`

### Shared request-context data plane

- Existing `/clinic/api/v1/data/*` routes continue using `RequestContext`

### Data Proxy

- `data-proxy/query`
- `data-proxy/schema`
- `data-proxy/loki/...`

These routes are cluster-ID keyed and require bearer auth, but do not use the shared `RequestContext` header model.

### NGM

`CloudNGMTarget` must carry:

- `TenantID`
- `ProjectID`
- `ClusterID`
- `DeployType`
- `Provider`
- `Region`

Premium / BYOC helpers must allow parent-cluster routing when the Cloud capability requires it.

## CLI Additions

Add these public commands:

- `clinic-client cloud cluster search`
- `clinic-client cloud cluster topology`
- `clinic-client cloud data-proxy query`
- `clinic-client cloud data-proxy schema`
- `clinic-client cloud loki query`
- `clinic-client cloud loki query-range`
- `clinic-client cloud loki labels`
- `clinic-client cloud loki label-values`
- `clinic-client cloud profiling groups`
- `clinic-client cloud profiling detail`
- `clinic-client cloud profiling download`
- `clinic-client cloud diagnostics plan-replayer`
- `clinic-client cloud diagnostics oom-record`
- `clinic-client cloud diagnostics download`

## Testing

- Endpoint coverage for each new SDK method
- Routing/helper coverage for Premium / BYOC / resource-pool rules
- CLI dispatch coverage for each new command family
- Output and download-path coverage where applicable
- Full `go test ./...` verification
