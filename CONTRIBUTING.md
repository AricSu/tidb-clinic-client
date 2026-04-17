# Contributing

## Development

Run the full test suite:

```bash
go test ./...
```

Format the repository before sending changes:

```bash
gofmt -w *.go
```

## Scope Discipline

This repository is the external Clinic data SDK.

Preferred public additions should land in the cluster-scoped facade layer:

- `Client.Clusters`
- `Client.Clusters.Resolve(...)->ClusterHandle`
- `ClusterHandle.Metrics`
- `ClusterHandle.Logs`
- `ClusterHandle.SQLAnalytics`
- `ClusterHandle.Configs`
- `ClusterHandle.Profiling`
- `ClusterHandle.Diagnostics`
- `ClusterHandle.Capabilities`

Transport and backend-shaped code may remain in-tree, but V2 changes should
flow through the cluster-scoped facade contract instead of reintroducing new
public low-level entry points.

Keep these concerns out of this repo:

- workflow orchestration
- diagnosis logic
- application-specific integration behavior

## Pull Request Expectations

- keep the public surface small
- prefer cluster-scoped public APIs over backend-shaped entry points
- add or update tests for behavior changes
- update README or examples when public usage changes
