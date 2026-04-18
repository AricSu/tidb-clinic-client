package clinicapi

import (
	"net/url"
	"path"
	"strings"
)

const portalURLParseEndpoint = "portal_url.parse"

// PortalURLInfo contains the cluster-scoped context encoded in a Clinic portal URL.
type PortalURLInfo struct {
	BaseURL   string
	ClusterID string
}

// ParsePortalURL extracts the Clinic base URL and cluster id from a Clinic portal URL.
func ParsePortalURL(raw string) (PortalURLInfo, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return PortalURLInfo{}, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: portalURLParseEndpoint,
			Message:  "portal URL is required",
		}
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return PortalURLInfo{}, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: portalURLParseEndpoint,
			Message:  "failed to parse portal URL",
			Cause:    err,
		}
	}
	if strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
		return PortalURLInfo{}, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: portalURLParseEndpoint,
			Message:  "portal URL must include scheme and host",
		}
	}

	route, err := portalRoute(parsed)
	if err != nil {
		return PortalURLInfo{}, err
	}
	clusterID := portalRouteClusterID(route.Path)
	if clusterID == "" {
		return PortalURLInfo{}, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: portalURLParseEndpoint,
			Message:  "portal URL must include /clusters/{clusterID}",
		}
	}

	return PortalURLInfo{
		BaseURL:   portalBaseURL(parsed),
		ClusterID: clusterID,
	}, nil
}

func portalRoute(parsed *url.URL) (*url.URL, error) {
	if parsed == nil {
		return nil, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: portalURLParseEndpoint,
			Message:  "portal URL is required",
		}
	}

	candidates := make([]string, 0, 2)
	if fragment := strings.TrimSpace(parsed.Fragment); fragment != "" {
		candidates = append(candidates, fragment)
	}
	if routePath := strings.TrimSpace(parsed.Path); routePath != "" && routePath != "/" {
		directRoute := routePath
		if rawQuery := strings.TrimSpace(parsed.RawQuery); rawQuery != "" {
			directRoute += "?" + rawQuery
		}
		candidates = append(candidates, directRoute)
	}

	for _, candidate := range candidates {
		route, err := url.Parse(strings.TrimPrefix(candidate, "#"))
		if err != nil {
			continue
		}
		if looksLikeClusterRoute(route.Path) {
			return route, nil
		}
	}

	return nil, &Error{
		Class:    ErrInvalidRequest,
		Endpoint: portalURLParseEndpoint,
		Message:  "portal URL must include a cluster route",
	}
}

func looksLikeClusterRoute(routePath string) bool {
	segments := routeSegments(routePath)
	hasCluster := false
	for i := 0; i < len(segments)-1; i++ {
		if segments[i] == "clusters" {
			hasCluster = true
		}
	}
	return hasCluster
}

func portalRouteClusterID(routePath string) string {
	segments := routeSegments(routePath)
	for i := 0; i < len(segments)-1; i++ {
		value, err := url.PathUnescape(strings.TrimSpace(segments[i+1]))
		if err != nil {
			value = strings.TrimSpace(segments[i+1])
		}
		if segments[i] == "clusters" {
			return value
		}
	}
	return ""
}

func routeSegments(routePath string) []string {
	trimmed := strings.Trim(strings.TrimSpace(routePath), "/")
	if trimmed == "" {
		return nil
	}
	rawSegments := strings.Split(trimmed, "/")
	segments := make([]string, 0, len(rawSegments))
	for _, segment := range rawSegments {
		segment = strings.TrimSpace(segment)
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return segments
}

func portalBaseURL(parsed *url.URL) string {
	base := url.URL{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
		Path:   portalBasePath(parsed.Path),
	}
	return strings.TrimRight(base.String(), "/")
}

func portalBasePath(rawPath string) string {
	cleaned := path.Clean("/" + strings.TrimSpace(rawPath))
	switch {
	case cleaned == "." || cleaned == "/":
		return ""
	case cleaned == "/portal":
		return ""
	case strings.HasSuffix(cleaned, "/portal"):
		return strings.TrimSuffix(cleaned, "/portal")
	case strings.Contains(cleaned, "/portal/"):
		prefix := cleaned[:strings.Index(cleaned, "/portal/")]
		if prefix == "/" {
			return ""
		}
		return prefix
	default:
		return cleaned
	}
}
