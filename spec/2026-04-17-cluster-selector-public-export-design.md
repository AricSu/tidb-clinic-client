# Cluster Selector Public Export Design

Superseded by `2026-04-17-single-cluster-id-public-surface-design.md`.

Extends the stable external model in
[`2026-04-16-capability-first-sdk-design.md`](./2026-04-16-capability-first-sdk-design.md)
by formalizing the advanced selector entry point in the root public surface.

## Summary

The cluster-scoped SDK already supports an internal advanced binding path:

- `Clusters.Resolve(clusterID)`
- `Clusters.ResolveWith(ClusterSelector{...})`

However, the root package has only exposed the canonical happy-path `Resolve`
model cleanly. Advanced callers such as workflow engines may still need a
public, typed escape hatch when they already hold authoritative cluster binding
metadata and want to re-bind deterministically without depending on internal
runtime packages.

This follow-up makes that advanced path first-class in the root package by
exporting `ClusterSelector` from the public surface and documenting
`ResolveWith(...)` as a supported but non-default entry point.

## Design Decisions

- `Resolve(clusterID)` remains the primary documented path.
- `ResolveWith(ClusterSelector)` is public and supported, but stays out of the
  default quick-start path.
- `ClusterSelector` contains only:
  - `ClusterID`
  - `Platform`
  - `OrgID`
- The selector is an advanced deterministic binding input, not a resolved
  target output model.
- SDK consumers should prefer `ResolveWith(...)` only when they already hold
  authoritative binding information and want to re-bind through the same
  cluster-scoped facade.

## Implementation Changes

- root `handle.go`
  - export `ClusterSelector` as a public alias
- root documentation
  - mention `ResolveWith(ClusterSelector)` as an advanced escape hatch
  - keep `Resolve(clusterID)` as the canonical happy path

## Non-Goals

- changing the default `Resolve(clusterID)` caller experience
- exposing internal resolved target models
- widening the selector with request-context or backend-specific fields
