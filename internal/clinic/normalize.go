package clinic

import (
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

func streamResultFromLoki(result apitypes.LokiQueryResult) StreamResult {
	out := StreamResult{
		Status:  result.Status,
		Streams: make([]Stream, 0, len(result.Streams)),
	}
	for _, stream := range result.Streams {
		next := Stream{
			Labels: cloneStringMap(stream.Labels),
			Values: make([]StreamValue, 0, len(stream.Values)),
		}
		for _, value := range stream.Values {
			next.Values = append(next.Values, StreamValue{
				Timestamp: value.Timestamp,
				Line:      value.Line,
			})
		}
		out.Streams = append(out.Streams, next)
	}
	return out
}

func listResultFromValues(values []string, status string) ListResult {
	items := make([]map[string]any, 0, len(values))
	for _, value := range values {
		items = append(items, map[string]any{"value": value})
	}
	return ListResult{
		Total: len(items),
		Items: items,
		Metadata: QueryMetadata{
			Raw: map[string]any{"status": status},
		},
	}
}

func listResultFromProfileGroups(result apitypes.CloudProfileGroupsResult) ListResult {
	items := make([]map[string]any, 0, len(result.Groups))
	for _, group := range result.Groups {
		componentNum := make(map[string]any, len(group.ComponentNum))
		for key, value := range group.ComponentNum {
			componentNum[key] = value
		}
		items = append(items, map[string]any{
			"timestamp":             group.Timestamp,
			"profile_duration_secs": group.ProfileDurationSecs,
			"state":                 group.State,
			"component_num":         componentNum,
			"raw":                   cloneAnyMap(group.Raw),
		})
	}
	return ListResult{Total: len(items), Items: items}
}

func listResultFromDiagnostics(result apitypes.CloudDiagnosticListResult) ListResult {
	items := make([]map[string]any, 0, len(result.Records))
	for _, record := range result.Records {
		files := make([]map[string]any, 0, len(record.Files))
		for _, file := range record.Files {
			files = append(files, map[string]any{
				"name":         file.Name,
				"key":          file.Key,
				"size":         file.Size,
				"download_url": file.DownloadURL,
				"raw":          cloneAnyMap(file.Raw),
			})
		}
		items = append(items, map[string]any{
			"files": files,
			"raw":   cloneAnyMap(record.Raw),
		})
	}
	return ListResult{Total: len(items), Items: items}
}
