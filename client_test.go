package clinicapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewClinicClientRejectsMissingBaseURL(t *testing.T) {
	_, err := NewClientWithConfig(Config{
		BearerToken: "token",
		Timeout:     5 * time.Second,
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "base") {
		t.Fatalf("expected base url validation error, got=%v", err)
	}
}

func TestNewClinicClientAcceptsAuthProviderWithoutBearerToken(t *testing.T) {
	client, err := NewClientWithConfig(Config{
		BaseURL:      "https://clinic.pingcap.com",
		AuthProvider: StaticBearerToken("token"),
		Timeout:      5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	if client == nil {
		t.Fatalf("expected client")
	}
}

func TestNewClinicClientInitializesDomainClients(t *testing.T) {
	client, err := NewClientWithConfig(Config{
		BaseURL:     "https://clinic.pingcap.com",
		BearerToken: "token",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	if client.Catalog == nil || client.Metrics == nil || client.SlowQueries == nil || client.Logs == nil || client.Configs == nil {
		t.Fatalf("expected all domain clients to be initialized: %+v", client)
	}
	if client.Cloud == nil {
		t.Fatalf("expected cloud client to be initialized: %+v", client)
	}
	if client.OP == nil {
		t.Fatalf("expected op placeholder client to be initialized: %+v", client)
	}
	if client.Config().BaseURL != "https://clinic.pingcap.com" {
		t.Fatalf("unexpected config snapshot: %+v", client.Config())
	}
}

func TestNewClinicClientNormalizesBearerTokenIntoAuthProvider(t *testing.T) {
	client, err := NewClientWithConfig(Config{
		BaseURL:     "https://clinic.pingcap.com",
		BearerToken: "token",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	cfg := client.Config()
	if cfg.BearerToken != "" {
		t.Fatalf("expected normalized config to clear bearer token, got=%q", cfg.BearerToken)
	}
	if cfg.AuthProvider == nil {
		t.Fatalf("expected normalized config to retain auth provider")
	}
}

func TestPublicSurfaceDoesNotExposeCapabilitiesAPI(t *testing.T) {
	clientType := reflect.TypeOf(&Client{})
	if _, ok := clientType.MethodByName("Capabilities"); ok {
		t.Fatalf("Client should not expose Capabilities method")
	}

	configType := reflect.TypeOf(Config{})
	if _, ok := configType.FieldByName("CapabilityResolver"); ok {
		t.Fatalf("Config should not expose CapabilityResolver field")
	}
}

func TestCatalogListClusterDataUsesAuthProvider(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total":     0,
			"dataInfos": []map[string]any{},
		})
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:      server.URL,
		AuthProvider: StaticBearerToken("provider-token"),
		Timeout:      5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.Catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err != nil {
		t.Fatalf("ListClusterData failed: %v", err)
	}
	if gotAuth != "Bearer provider-token" {
		t.Fatalf("unexpected auth header from provider: %q", gotAuth)
	}
}

func TestAuthProviderOverridesBearerToken(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total":     0,
			"dataInfos": []map[string]any{},
		})
	}))
	defer server.Close()

	provider := AuthProviderFunc(func(req *http.Request) error {
		req.Header.Set("Authorization", "Bearer provider-wins")
		return nil
	})
	client, err := NewClientWithConfig(Config{
		BaseURL:      server.URL,
		BearerToken:  "config-token",
		AuthProvider: provider,
		Timeout:      5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.Catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err != nil {
		t.Fatalf("ListClusterData failed: %v", err)
	}
	if gotAuth != "Bearer provider-wins" {
		t.Fatalf("expected auth provider to override bearer token, got=%q", gotAuth)
	}
}

func TestTransportClassifiesAuthProviderFailure(t *testing.T) {
	providerErr := errors.New("missing credential")
	client, err := NewClientWithConfig(Config{
		BaseURL: "https://clinic.pingcap.com",
		AuthProvider: AuthProviderFunc(func(req *http.Request) error {
			return providerErr
		}),
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.Catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err == nil {
		t.Fatalf("expected auth provider failure")
	}
	clinicErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T err=%v", err, err)
	}
	if clinicErr.Class != ErrAuth || clinicErr.Retryable {
		t.Fatalf("unexpected auth provider classification: %+v", clinicErr)
	}
	if !strings.Contains(clinicErr.Message, "failed to apply clinic auth") {
		t.Fatalf("unexpected auth provider message: %q", clinicErr.Message)
	}
}

func TestRouteFromContextBuildsHeadersAndTrace(t *testing.T) {
	route, err := routeFromContext(metricsEndpoint, RequestContext{
		OrgType:   "cloud",
		OrgID:     "org-1",
		ClusterID: "cluster-9",
	}, "item-1")
	if err != nil {
		t.Fatalf("routeFromContext failed: %v", err)
	}
	if route.headers.Get("X-OrgType") != "cloud" || route.headers.Get("X-OrgID") != "org-1" || route.headers.Get("X-ClusterID") != "cluster-9" {
		t.Fatalf("unexpected routing headers: %+v", route.headers)
	}
	if route.trace.orgType != "cloud" || route.trace.orgID != "org-1" || route.trace.clusterID != "cluster-9" || route.trace.itemID != "item-1" {
		t.Fatalf("unexpected route trace: %+v", route.trace)
	}
}

func TestRouteFromItemContextRequiresItemID(t *testing.T) {
	_, err := routeFromItemContext(slowQueriesEndpoint, RequestContext{
		OrgType:   "cloud",
		OrgID:     "org-1",
		ClusterID: "cluster-9",
	}, "")
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if ClassOf(err) != ErrInvalidRequest {
		t.Fatalf("unexpected error class: %v", err)
	}
}

func TestRouteFromCloudTargetUsesCloudTraceWithoutHeaders(t *testing.T) {
	route, err := routeFromCloudTarget(clusterDetailPattern, CloudTarget{
		OrgID:     "org-1",
		ClusterID: "cluster-9",
	})
	if err != nil {
		t.Fatalf("routeFromCloudTarget failed: %v", err)
	}
	if len(route.headers) != 0 {
		t.Fatalf("expected no extra headers, got %+v", route.headers)
	}
	if route.trace.orgType != "cloud" || route.trace.orgID != "org-1" || route.trace.clusterID != "cluster-9" {
		t.Fatalf("unexpected route trace: %+v", route.trace)
	}
}

func TestRouteFromCloudNGMTargetBuildsNGMHeadersAndTrace(t *testing.T) {
	route, err := routeFromCloudNGMTarget(ngmTopSQLEndpoint, CloudNGMTarget{
		Provider:   "aws",
		Region:     "us-east-1",
		TenantID:   "tenant-1",
		ProjectID:  "project-1",
		ClusterID:  "cluster-9",
		DeployType: "dedicated",
	})
	if err != nil {
		t.Fatalf("routeFromCloudNGMTarget failed: %v", err)
	}
	if route.headers.Get("X-Provider") != "aws" ||
		route.headers.Get("X-Region") != "us-east-1" ||
		route.headers.Get("X-Org-Id") != "tenant-1" ||
		route.headers.Get("X-Project-Id") != "project-1" ||
		route.headers.Get("X-Cluster-Id") != "cluster-9" ||
		route.headers.Get("X-Deploy-Type") != "dedicated" {
		t.Fatalf("unexpected ngm headers: %+v", route.headers)
	}
	if route.trace.orgType != "cloud" || route.trace.orgID != "tenant-1" || route.trace.clusterID != "cluster-9" {
		t.Fatalf("unexpected route trace: %+v", route.trace)
	}
}

func TestRouteHelpersClassifyInvalidInput(t *testing.T) {
	_, err := routeFromCloudNGMTarget(ngmTopSQLEndpoint, CloudNGMTarget{
		Provider:   "aws",
		Region:     "us-east-1",
		TenantID:   "",
		ProjectID:  "project-1",
		ClusterID:  "cluster-9",
		DeployType: "dedicated",
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if ClassOf(err) != ErrInvalidRequest {
		t.Fatalf("unexpected error class: %v", err)
	}
}

func TestRouteFromKnownClusterBuildsTraceAndClonedHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Test", "value")
	route, err := routeFromKnownCluster(metricsEndpoint, knownClusterRouteInput{
		orgType:   "op",
		orgID:     "org-1",
		clusterID: "cluster-9",
		itemID:    "item-1",
		headers:   headers,
	})
	if err != nil {
		t.Fatalf("routeFromKnownCluster failed: %v", err)
	}
	if route.trace.orgType != "op" || route.trace.orgID != "org-1" || route.trace.clusterID != "cluster-9" || route.trace.itemID != "item-1" {
		t.Fatalf("unexpected route trace: %+v", route.trace)
	}
	if route.headers.Get("X-Test") != "value" {
		t.Fatalf("unexpected route headers: %+v", route.headers)
	}
	headers.Set("X-Test", "mutated")
	if route.headers.Get("X-Test") != "value" {
		t.Fatalf("expected route headers to be cloned, got %+v", route.headers)
	}
}

func TestRouteFromKnownClusterClassifiesInvalidInput(t *testing.T) {
	_, err := routeFromKnownCluster(metricsEndpoint, knownClusterRouteInput{
		orgType:   "",
		orgID:     "org-1",
		clusterID: "cluster-9",
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if ClassOf(err) != ErrInvalidRequest {
		t.Fatalf("unexpected error class: %v", err)
	}
}
