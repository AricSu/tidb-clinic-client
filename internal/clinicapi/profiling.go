package clinicapi

import (
	"context"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	continuousProfilingListPath        = ngmContinuousProfilingBase + "/list"
	continuousProfilingDetailPath      = ngmContinuousProfilingBase + "/detail"
	continuousProfilingActionTokenPath = ngmContinuousProfilingBase + "/action_token"
	continuousProfilingDownloadPath    = ngmContinuousProfilingBase + "/download"
)

type CloudProfileGroupsRequest struct {
	Target    CloudNGMTarget
	BeginTime int64
	EndTime   int64
}
type CloudProfileGroup struct {
	Timestamp           int64
	ProfileDurationSecs int64
	State               string
	ComponentNum        map[string]int64
	Raw                 map[string]any
}
type CloudProfileGroupsResult struct {
	Groups []CloudProfileGroup
}
type CloudProfileGroupDetailRequest struct {
	Target    CloudNGMTarget
	Timestamp int64
}
type CloudProfileTargetProfile struct {
	ProfileType string
	State       string
	Error       string
	Component   string
	Address     string
	Raw         map[string]any
}
type CloudProfileGroupDetail struct {
	TargetProfiles []CloudProfileTargetProfile
	Raw            map[string]any
}
type CloudProfileActionTokenRequest struct {
	Target      CloudNGMTarget
	Timestamp   int64
	ProfileType string
	Component   string
	Address     string
	DataFormat  string
}
type CloudProfileDownloadRequest struct {
	Target CloudNGMTarget
	Token  string
}
type CloudProfileFetchRequest struct {
	Target      CloudNGMTarget
	Timestamp   int64
	ProfileType string
	Component   string
	Address     string
	DataFormat  string
}
type CloudDownloadedArtifact struct {
	Filename    string
	ContentType string
	Bytes       []byte
}

func (c *cloudClient) ListProfileGroups(ctx context.Context, req CloudProfileGroupsRequest) (CloudProfileGroupsResult, error) {
	if c == nil || c.transport == nil {
		return CloudProfileGroupsResult{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	if req.BeginTime <= 0 || req.EndTime <= 0 || req.EndTime < req.BeginTime {
		return CloudProfileGroupsResult{}, &Error{Class: ErrInvalidRequest, Endpoint: continuousProfilingListPath, Message: "valid begin/end range is required"}
	}
	route, err := routeFromCloudNGMTarget(continuousProfilingListPath, req.Target)
	if err != nil {
		return CloudProfileGroupsResult{}, err
	}
	query := url.Values{}
	query.Set("begin_time", strconv.FormatInt(req.BeginTime, 10))
	query.Set("end_time", strconv.FormatInt(req.EndTime, 10))
	var raw any
	if err := c.transport.getJSON(ctx, continuousProfilingListPath, query, route.headers, route.trace, &raw); err != nil {
		return CloudProfileGroupsResult{}, err
	}
	return normalizeProfileGroups(raw), nil
}
func (c *cloudClient) GetProfileGroupDetail(ctx context.Context, req CloudProfileGroupDetailRequest) (CloudProfileGroupDetail, error) {
	if c == nil || c.transport == nil {
		return CloudProfileGroupDetail{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	if req.Timestamp <= 0 {
		return CloudProfileGroupDetail{}, &Error{Class: ErrInvalidRequest, Endpoint: continuousProfilingDetailPath, Message: "timestamp is required"}
	}
	route, err := routeFromCloudNGMTarget(continuousProfilingDetailPath, req.Target)
	if err != nil {
		return CloudProfileGroupDetail{}, err
	}
	query := url.Values{}
	query.Set("timestamp", strconv.FormatInt(req.Timestamp, 10))
	var raw map[string]any
	if err := c.transport.getJSON(ctx, continuousProfilingDetailPath, query, route.headers, route.trace, &raw); err != nil {
		return CloudProfileGroupDetail{}, err
	}
	return normalizeProfileGroupDetail(raw), nil
}
func (c *cloudClient) GetProfileActionToken(ctx context.Context, req CloudProfileActionTokenRequest) (string, error) {
	if c == nil || c.transport == nil {
		return "", &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	if req.Timestamp <= 0 {
		return "", &Error{Class: ErrInvalidRequest, Endpoint: continuousProfilingActionTokenPath, Message: "timestamp is required"}
	}
	if strings.TrimSpace(req.ProfileType) == "" {
		return "", &Error{Class: ErrInvalidRequest, Endpoint: continuousProfilingActionTokenPath, Message: "profile type is required"}
	}
	route, err := routeFromCloudNGMTarget(continuousProfilingActionTokenPath, req.Target)
	if err != nil {
		return "", err
	}
	query := url.Values{}
	query.Set("timestamp", strconv.FormatInt(req.Timestamp, 10))
	query.Set("profile_type", strings.TrimSpace(req.ProfileType))
	if component := strings.TrimSpace(req.Component); component != "" {
		query.Set("component", component)
	}
	if address := strings.TrimSpace(req.Address); address != "" {
		query.Set("address", address)
	}
	if dataFormat := strings.TrimSpace(req.DataFormat); dataFormat != "" {
		query.Set("data_format", dataFormat)
	}
	var raw any
	if err := c.transport.getJSON(ctx, continuousProfilingActionTokenPath, query, route.headers, route.trace, &raw); err != nil {
		return "", err
	}
	switch x := raw.(type) {
	case string:
		return strings.TrimSpace(x), nil
	case map[string]any:
		return asTrimmedString(firstPresent(x, "token", "action_token", "data")), nil
	default:
		return "", nil
	}
}
func (c *cloudClient) DownloadProfile(ctx context.Context, req CloudProfileDownloadRequest) (CloudDownloadedArtifact, error) {
	if c == nil || c.transport == nil {
		return CloudDownloadedArtifact{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	if strings.TrimSpace(req.Token) == "" {
		return CloudDownloadedArtifact{}, &Error{Class: ErrInvalidRequest, Endpoint: continuousProfilingDownloadPath, Message: "token is required"}
	}
	route, err := routeFromCloudNGMTarget(continuousProfilingDownloadPath, req.Target)
	if err != nil {
		return CloudDownloadedArtifact{}, err
	}
	query := url.Values{}
	query.Set("token", strings.TrimSpace(req.Token))
	body, err := c.transport.getBytes(ctx, continuousProfilingDownloadPath, query, route.headers, route.trace)
	if err != nil {
		return CloudDownloadedArtifact{}, err
	}
	return CloudDownloadedArtifact{
		Filename: defaultProfileDownloadName(req.Token),
		Bytes:    body,
	}, nil
}
func (c *cloudClient) FetchProfile(ctx context.Context, req CloudProfileFetchRequest) (CloudDownloadedArtifact, error) {
	token, err := c.GetProfileActionToken(ctx, CloudProfileActionTokenRequest{
		Target:      req.Target,
		Timestamp:   req.Timestamp,
		ProfileType: req.ProfileType,
		Component:   req.Component,
		Address:     req.Address,
		DataFormat:  req.DataFormat,
	})
	if err != nil {
		return CloudDownloadedArtifact{}, err
	}
	return c.DownloadProfile(ctx, CloudProfileDownloadRequest{
		Target: req.Target,
		Token:  token,
	})
}
func normalizeProfileGroups(raw any) CloudProfileGroupsResult {
	_, groups := unwrapCollection(raw)
	out := CloudProfileGroupsResult{Groups: make([]CloudProfileGroup, 0, len(groups))}
	for _, groupRaw := range groups {
		group := CloudProfileGroup{
			Timestamp:           asInt64OrZero(firstPresent(groupRaw, "timestamp", "ts")),
			ProfileDurationSecs: asInt64OrZero(firstPresent(groupRaw, "profile_duration_secs", "profileDurationSecs", "duration")),
			State:               asTrimmedString(firstPresent(groupRaw, "state", "status")),
			ComponentNum:        make(map[string]int64),
			Raw:                 cloneAnyMap(groupRaw),
		}
		componentNum := asAnyMap(firstPresent(groupRaw, "component_num", "componentNum"))
		for key, value := range componentNum {
			group.ComponentNum[key] = asInt64OrZero(value)
		}
		out.Groups = append(out.Groups, group)
	}
	return out
}
func normalizeProfileGroupDetail(raw map[string]any) CloudProfileGroupDetail {
	out := CloudProfileGroupDetail{Raw: cloneAnyMap(raw)}
	for _, item := range sliceOfMaps(firstPresent(raw, "target_profiles", "targetProfiles", "profiles", "data")) {
		out.TargetProfiles = append(out.TargetProfiles, CloudProfileTargetProfile{
			ProfileType: asTrimmedString(firstPresent(item, "profile_type", "profileType")),
			State:       asTrimmedString(firstPresent(item, "state", "status")),
			Error:       asTrimmedString(item["error"]),
			Component:   asTrimmedString(item["component"]),
			Address:     asTrimmedString(firstPresent(item, "address", "instance")),
			Raw:         cloneAnyMap(item),
		})
	}
	return out
}
func defaultProfileDownloadName(token string) string {
	base := filepath.Base(strings.TrimSpace(token))
	if base == "" || base == "." || base == ".." {
		return "profile.data"
	}
	return base
}
