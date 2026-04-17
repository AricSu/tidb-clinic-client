package clinicapi

import (
	"context"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

func (c *cloudClient) ListPlanReplayerArtifacts(ctx context.Context, req CloudDiagnosticListRequest) (CloudDiagnosticListResult, error) {
	return c.listDiagnosticArtifacts(ctx, ngmPlanReplayerListPath, req)
}
func (c *cloudClient) ListOOMRecordArtifacts(ctx context.Context, req CloudDiagnosticListRequest) (CloudDiagnosticListResult, error) {
	return c.listDiagnosticArtifacts(ctx, ngmOOMRecordListPath, req)
}
func (c *cloudClient) DownloadDiagnosticArtifact(ctx context.Context, req CloudDiagnosticDownloadRequest) (CloudDownloadedArtifact, error) {
	if c == nil || c.transport == nil {
		return CloudDownloadedArtifact{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	if strings.TrimSpace(req.Key) == "" {
		return CloudDownloadedArtifact{}, &Error{Class: ErrInvalidRequest, Endpoint: ngmOOMRecordFilesPath, Message: "key is required"}
	}
	route, err := routeFromCloudNGMTarget(ngmOOMRecordFilesPath, req.Target)
	if err != nil {
		return CloudDownloadedArtifact{}, err
	}
	query := url.Values{}
	query.Set("provider", strings.TrimSpace(req.Target.Provider))
	query.Set("region", strings.TrimSpace(req.Target.Region))
	query.Set("key", strings.TrimSpace(req.Key))
	body, err := c.transport.getBytes(ctx, ngmOOMRecordFilesPath, query, route.headers, route.trace)
	if err != nil {
		return CloudDownloadedArtifact{}, err
	}
	return CloudDownloadedArtifact{
		Filename: defaultDiagnosticDownloadName(req.Key),
		Bytes:    body,
	}, nil
}
func (c *cloudClient) listDiagnosticArtifacts(ctx context.Context, endpoint string, req CloudDiagnosticListRequest) (CloudDiagnosticListResult, error) {
	if c == nil || c.transport == nil {
		return CloudDiagnosticListResult{}, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	if req.Start <= 0 || req.End <= 0 || req.End < req.Start {
		return CloudDiagnosticListResult{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "valid start/end range is required"}
	}
	route, err := routeFromCloudNGMTarget(endpoint, req.Target)
	if err != nil {
		return CloudDiagnosticListResult{}, err
	}
	query := url.Values{}
	query.Set("start", strconv.FormatInt(req.Start, 10))
	query.Set("end", strconv.FormatInt(req.End, 10))
	var raw any
	if err := c.transport.getJSON(ctx, endpoint, query, route.headers, route.trace, &raw); err != nil {
		return CloudDiagnosticListResult{}, err
	}
	return normalizeDiagnosticList(raw), nil
}
func normalizeDiagnosticList(raw any) CloudDiagnosticListResult {
	var groups []map[string]any
	switch x := raw.(type) {
	case map[string]any:
		switch {
		case x["records"] != nil:
			groups = sliceOfMaps(x["records"])
		case x["data"] != nil:
			groups = sliceOfMaps(x["data"])
		default:
			groups = []map[string]any{x}
		}
	case []any:
		groups = sliceOfMaps(x)
	}
	out := CloudDiagnosticListResult{Records: make([]CloudDiagnosticRecordGroup, 0, len(groups))}
	for _, groupRaw := range groups {
		group := CloudDiagnosticRecordGroup{
			Raw: cloneAnyMap(groupRaw),
		}
		files := sliceOfMaps(firstPresent(groupRaw, "files", "artifacts", "items"))
		for _, fileRaw := range files {
			group.Files = append(group.Files, CloudDiagnosticFile{
				Name:        asTrimmedString(firstPresent(fileRaw, "name", "file_name", "filename")),
				Key:         asTrimmedString(fileRaw["key"]),
				Size:        asInt64OrZero(firstPresent(fileRaw, "size", "file_size")),
				DownloadURL: asTrimmedString(firstPresent(fileRaw, "download_url", "downloadURL")),
				Raw:         cloneAnyMap(fileRaw),
			})
		}
		out.Records = append(out.Records, group)
	}
	return out
}
func defaultDiagnosticDownloadName(key string) string {
	base := filepath.Base(strings.TrimSpace(key))
	if base == "" || base == "." || base == ".." {
		return "diagnostic.dat"
	}
	return base
}
