# Single Cluster-ID Public Surface Design

This design supersedes the advanced public selector proposal in:

- `2026-04-17-cluster-selector-public-export-design.md`

It also narrows the public shape introduced by:

- `2026-04-16-capability-first-sdk-design.md`

## Summary

The public SDK surface converges on one binding entry only:

- `client.Clusters.Resolve(ctx, clusterID) -> ClusterHandle`

The SDK still keeps richer selector-like runtime models internally, but those
models are implementation details and must not be exposed from the root package.

## Public Rules

- callers provide only `cluster_id`
- SDK resolves `org_id`, cluster metadata, and platform automatically
- no public `ResolveWith(...)`
- no public `ClusterSelector`
- no public requirement for callers to pass `platform`
- no public requirement for callers to pass `org_id`

## Resolve Semantics

`Resolve(clusterID)` remains authoritative and deterministic:

- `0` matches: `ErrNotFound`
- `1` match: return `ClusterHandle`
- `>1` matches: `ErrInvalidRequest`

The caller does not participate in ambiguity resolution through a second
binding API.

## Internal Rules

Internal runtime packages may still use selector-like models for:

- backend lookups
- normalization
- capability binding
- resolved identity caching

But those types stay under `internal/` and do not leak through the root
package.

## Retained-Data Contract

The same simplification applies to OP retained-data behavior:

- callers should not need to pass `item_id` to keep records / sample / detail
  stable
- retained-data continuity is an SDK guarantee
- provenance is exposed through a single typed `RetainedDataRef` result field
- legacy compatibility fields such as top-level `SourceRef` / `ItemID` on public
  result models are retired
- retained-data provenance is observable, but not required as caller input

## Implementation Changes

- root package stops exporting `ClusterSelector`
- cluster facade stops exposing public `ResolveWith(...)`
- docs and README describe only the cluster-id-first flow
- public result models converge on `RetainedDataRef` instead of duplicated
  compatibility fields
- retained-data plumbing stays internal to capability implementations

## Non-Goals

- changing internal runtime resolve machinery
- removing `TargetPlatform` from result metadata such as `ClusterHandle.Platform()`
- moving analysis/orchestration logic out of downstream workflow consumers
