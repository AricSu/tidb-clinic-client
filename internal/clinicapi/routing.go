package clinicapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	clusterLookupPath        = apiV1Prefix + "/dashboard/clusters"
	metricsEndpoint          = apiV1Prefix + "/data/metrics"
	slowQueriesEndpoint      = apiV1Prefix + "/data/slowqueries"
	logsEndpoint             = apiV1Prefix + "/data/logs"
	configEndpoint           = apiV1Prefix + "/data/config"
	dataProxyQueryPath       = "/data-proxy/query"
	dataProxySchemaPath      = "/data-proxy/schema"
	lokiEndpointPrefix       = "/data-proxy/loki"
	orgDetailPattern         = apiV1Prefix + "/orgs/{orgID}"
	catalogEndpointPattern   = apiV1Prefix + "/orgs/{orgID}/clusters/{clusterID}/data"
	catalogDataStatusPattern = apiV1Prefix + "/orgs/{orgID}/clusters/{clusterID}/data_status"
	catalogRebuildPattern    = apiV1Prefix + "/orgs/{orgID}/clusters/{clusterID}/rebuild"
	clusterDetailPattern     = apiV1Prefix + "/orgs/{orgID}/clusters/{clusterID}"
	collectedSlowQueriesPath = apiV1Prefix + "/orgs/{orgID}/clusters/{clusterID}/slowqueries"
	collectedSlowQueryDetail = apiV1Prefix + "/orgs/{orgID}/clusters/{clusterID}/slowqueries/{slowQueryID}"
	cloudEventsPattern       = apiV1Prefix + "/activityhub/applications/{orgID}/targets/{clusterID}/activities"
	ngmTopSQLEndpoint        = ngmAPIV1Prefix + "/topsql/summary"
	ngmTopSlowQueriesPath    = ngmAPIV1Prefix + "/slow_query/stats"
	ngmSlowQueryListPath     = ngmAPIV1Prefix + "/slow_query/list"
	ngmSlowQueryDetailPath   = ngmAPIV1Prefix + "/slow_query/detail"
	ngmPlanReplayerListPath  = ngmAPIV1Prefix + "/plan_replayer/list"
	ngmOOMRecordListPath     = ngmAPIV1Prefix + "/oom_record/list"
	ngmOOMRecordFilesPath    = ngmAPIV1Prefix + "/oom_record/files"
)
const maxSamplesErrorMessage = "cannot select more than -search.maxSamplesPerQuery"

type requestRoute struct {
	headers http.Header
	trace   requestTrace
}
type knownClusterRouteInput struct {
	orgType   string
	orgID     string
	clusterID string
	itemID    string
	headers   http.Header
}

func catalogEndpoint(orgID, clusterID string) string {
	return fmt.Sprintf(
		"%s/orgs/%s/clusters/%s/data",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
	)
}
func catalogDataStatusEndpoint(orgID, clusterID string) string {
	return fmt.Sprintf(
		"%s/orgs/%s/clusters/%s/data_status",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
	)
}
func catalogRebuildEndpoint(orgID, clusterID string) string {
	return fmt.Sprintf(
		"%s/orgs/%s/clusters/%s/rebuild",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
	)
}
func clusterLookupEndpoint() string {
	return clusterLookupPath
}
func clusterDetailEndpoint(orgID, clusterID string) string {
	return fmt.Sprintf(
		"%s/orgs/%s/clusters/%s",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
	)
}
func collectedSlowQueriesEndpoint(orgID, clusterID string) string {
	return fmt.Sprintf(
		"%s/orgs/%s/clusters/%s/slowqueries",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
	)
}
func collectedSlowQueryDetailEndpoint(orgID, clusterID, slowQueryID string) string {
	return fmt.Sprintf(
		"%s/orgs/%s/clusters/%s/slowqueries/%s",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
		url.PathEscape(strings.TrimSpace(slowQueryID)),
	)
}
func orgDetailEndpoint(orgID string) string {
	return fmt.Sprintf(
		"%s/orgs/%s",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
	)
}
func cloudEventsEndpoint(orgID, clusterID string) string {
	return fmt.Sprintf(
		"%s/activityhub/applications/%s/targets/%s/activities",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
	)
}
func cloudEventDetailEndpoint(orgID, clusterID, eventID string) string {
	return fmt.Sprintf(
		"%s/activityhub/applications/%s/targets/%s/activities/%s",
		apiV1Prefix,
		url.PathEscape(strings.TrimSpace(orgID)),
		url.PathEscape(strings.TrimSpace(clusterID)),
		url.PathEscape(strings.TrimSpace(eventID)),
	)
}
func validateCloudTarget(endpoint string, target CloudTarget) error {
	if strings.TrimSpace(target.OrgID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "org id is required"}
	}
	if strings.TrimSpace(target.ClusterID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "cluster id is required"}
	}
	return nil
}
func validateClusterLookup(endpoint string, req CloudClusterLookupRequest) error {
	if strings.TrimSpace(req.ClusterID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "cluster id is required"}
	}
	return nil
}
func validateCloudNGMTarget(endpoint string, target CloudNGMTarget) error {
	if strings.TrimSpace(target.Provider) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "provider is required"}
	}
	if strings.TrimSpace(target.Region) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "region is required"}
	}
	if strings.TrimSpace(target.TenantID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "tenant id is required"}
	}
	if strings.TrimSpace(target.ProjectID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "project id is required"}
	}
	if strings.TrimSpace(target.ClusterID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "cluster id is required"}
	}
	if strings.TrimSpace(target.DeployType) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "deploy type is required"}
	}
	return nil
}
func routingHeaders(ctx RequestContext) http.Header {
	headers := http.Header{}
	headers.Set("X-OrgType", strings.TrimSpace(ctx.RoutingOrgType))
	headers.Set("X-OrgID", strings.TrimSpace(ctx.OrgID))
	headers.Set("X-ClusterID", strings.TrimSpace(ctx.ClusterID))
	return headers
}
func ngmHeaders(target CloudNGMTarget) http.Header {
	headers := http.Header{}
	headers.Set("X-Provider", strings.TrimSpace(target.Provider))
	headers.Set("X-Region", strings.TrimSpace(target.Region))
	headers.Set("X-Org-Id", strings.TrimSpace(target.TenantID))
	headers.Set("X-Project-Id", strings.TrimSpace(target.ProjectID))
	headers.Set("X-Cluster-Id", strings.TrimSpace(target.ClusterID))
	headers.Set("X-Deploy-Type", strings.TrimSpace(target.DeployType))
	return headers
}
func requestTraceFromContext(ctx RequestContext, itemID string) requestTrace {
	return requestTrace{
		orgType:   strings.TrimSpace(ctx.OrgType),
		orgID:     strings.TrimSpace(ctx.OrgID),
		clusterID: strings.TrimSpace(ctx.ClusterID),
		itemID:    strings.TrimSpace(itemID),
	}
}
func routeFromKnownCluster(endpoint string, input knownClusterRouteInput) (requestRoute, error) {
	if strings.TrimSpace(input.orgType) == "" {
		return requestRoute{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "org type is required"}
	}
	if strings.TrimSpace(input.orgID) == "" {
		return requestRoute{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "org id is required"}
	}
	if strings.TrimSpace(input.clusterID) == "" {
		return requestRoute{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "cluster id is required"}
	}
	return requestRoute{
		headers: cloneHeader(input.headers),
		trace: requestTrace{
			orgType:   strings.TrimSpace(input.orgType),
			orgID:     strings.TrimSpace(input.orgID),
			clusterID: strings.TrimSpace(input.clusterID),
			itemID:    strings.TrimSpace(input.itemID),
		},
	}, nil
}
func routeFromContext(endpoint string, ctx RequestContext, itemID string) (requestRoute, error) {
	if strings.TrimSpace(ctx.RoutingOrgType) == "" {
		return requestRoute{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "routing org type is required"}
	}
	if strings.TrimSpace(ctx.OrgID) == "" {
		return requestRoute{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "org id is required"}
	}
	if strings.TrimSpace(ctx.ClusterID) == "" {
		return requestRoute{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "cluster id is required"}
	}
	return requestRoute{
		headers: routingHeaders(ctx),
		trace: requestTrace{
			orgType:   strings.TrimSpace(ctx.OrgType),
			orgID:     strings.TrimSpace(ctx.OrgID),
			clusterID: strings.TrimSpace(ctx.ClusterID),
			itemID:    strings.TrimSpace(itemID),
		},
	}, nil
}
func routeFromItemContext(endpoint string, ctx RequestContext, itemID string) (requestRoute, error) {
	if strings.TrimSpace(itemID) == "" {
		return requestRoute{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "item id is required"}
	}
	return routeFromContext(endpoint, ctx, itemID)
}
func routeFromCloudTarget(endpoint string, target CloudTarget) (requestRoute, error) {
	if err := validateCloudTarget(endpoint, target); err != nil {
		return requestRoute{}, err
	}
	return routeFromKnownCluster(endpoint, knownClusterRouteInput{
		orgType:   "cloud",
		orgID:     target.OrgID,
		clusterID: target.ClusterID,
	})
}
func routeFromSharedControlPlaneTarget(endpoint string, target CloudTarget, traceOrgType string) (requestRoute, error) {
	if err := validateCloudTarget(endpoint, target); err != nil {
		return requestRoute{}, err
	}
	return requestRoute{
		trace: requestTrace{
			orgType:   strings.TrimSpace(traceOrgType),
			orgID:     strings.TrimSpace(target.OrgID),
			clusterID: strings.TrimSpace(target.ClusterID),
		},
	}, nil
}
func routeFromClusterLookup(endpoint string, req CloudClusterLookupRequest) (requestRoute, error) {
	if err := validateClusterLookup(endpoint, req); err != nil {
		return requestRoute{}, err
	}
	return requestRoute{
		trace: requestTrace{
			clusterID: strings.TrimSpace(req.ClusterID),
		},
	}, nil
}
func routeFromCloudNGMTarget(endpoint string, target CloudNGMTarget) (requestRoute, error) {
	if err := validateCloudNGMTarget(endpoint, target); err != nil {
		return requestRoute{}, err
	}
	return routeFromKnownCluster(endpoint, knownClusterRouteInput{
		orgType:   "cloud",
		orgID:     target.TenantID,
		clusterID: target.ClusterID,
		headers:   ngmHeaders(target),
	})
}
func validateClusterIDOnly(endpoint, clusterID string) error {
	if strings.TrimSpace(clusterID) == "" {
		return &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "cluster id is required"}
	}
	return nil
}
func routeFromClusterIDOnly(endpoint, clusterID string) (requestRoute, error) {
	if err := validateClusterIDOnly(endpoint, clusterID); err != nil {
		return requestRoute{}, err
	}
	return requestRoute{
		trace: requestTrace{
			clusterID: strings.TrimSpace(clusterID),
		},
	}, nil
}
func lokiQueryEndpoint(clusterID string) string {
	return strings.TrimRight(lokiEndpointPrefix, "/") + "/" + strings.TrimSpace(clusterID) + "/api/v1/query"
}
func lokiQueryRangeEndpoint(clusterID string) string {
	return strings.TrimRight(lokiEndpointPrefix, "/") + "/" + strings.TrimSpace(clusterID) + "/api/v1/query_range"
}
func lokiLabelsEndpoint(clusterID string) string {
	return strings.TrimRight(lokiEndpointPrefix, "/") + "/" + strings.TrimSpace(clusterID) + "/api/v1/labels"
}
func lokiLabelValuesEndpoint(clusterID, labelName string) string {
	return strings.TrimRight(lokiEndpointPrefix, "/") + "/" + strings.TrimSpace(clusterID) + "/api/v1/label/" + strings.TrimSpace(labelName) + "/values"
}
func asInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int:
		return int64(x), true
	case float64:
		return int64(x), true
	case json.Number:
		n, err := x.Int64()
		if err == nil {
			return n, true
		}
		f, err := x.Float64()
		return int64(f), err == nil
	case jsonNumber:
		n, err := strconv.ParseInt(string(x), 10, 64)
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
func asFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		n, err := x.Float64()
		return n, err == nil
	case jsonNumber:
		n, err := strconv.ParseFloat(string(x), 64)
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return n, err == nil
	default:
		return 0, false
	}
}

type jsonNumber string

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func cloneHeader(in http.Header) http.Header {
	if len(in) == 0 {
		return nil
	}
	out := make(http.Header, len(in))
	for key, values := range in {
		out[key] = append([]string(nil), values...)
	}
	return out
}
