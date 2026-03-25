// Package clinicapi provides a typed Go client for the TiDB Clinic API.
//
// The package is intentionally scoped to transport and endpoint concerns:
//
//   - client construction and configuration
//   - auth and routing headers
//   - retry and timeout behavior
//   - typed request and response models
//   - stable error classification
//   - request lifecycle logging and hooks
//
// It does not implement higher-level fetch planning, item selection, anomaly
// observation, or diagnosis logic. Those behaviors belong in calling
// applications and workflow layers built on top of this package.
//
// The client currently supports typed access to:
//
//   - cloud cluster lookup by known cluster identifier
//   - uploaded data catalog
//   - metrics range queries
//   - metrics range queries with automatic time splitting on max-samples errors
//   - slow query retrieval
//   - log search
//   - config snapshots
//   - known-target cloud cluster detail
//   - known-target cloud events and event detail
//   - known-target cloud NGM TopSQL summary
//   - known-target cloud NGM slow-query summary, list, and detail
//   - a reserved OP sub-client placeholder for future On-Premise-only APIs
//
// Authentication can be configured either with the convenience BearerToken
// field or with a custom AuthProvider for rotating credentials and custom
// signing logic. AuthProvider is the normalized runtime model; bearer tokens
// are retained only as a construction convenience.
//
// By default, requests identify themselves with User-Agent:
//
//	tidb-clinic-client
//
// Callers can override that value when they need custom attribution.
//
// The package intentionally assumes a known target cluster. Broad discovery
// control-plane APIs such as cluster inventory browsing are outside the core
// SDK surface, but cloud integrations can resolve routing metadata from a
// known cluster identifier with Client.Cloud.GetCluster. The cloud NGM APIs
// still require explicit provider, region, tenant, project, cluster, and
// deploy-type routing values; CloudCluster helper methods make those values
// derivable from the lookup result. Current On-Premise support continues to
// live in the shared RequestContext-based APIs; Client.OP is currently only a
// reserved namespace for future On-Premise-only methods.
//
// The package is designed as a small product-grade SDK surface: stable
// construction, explicit request and response types, typed errors, and opt-in
// observability.
package clinicapi
