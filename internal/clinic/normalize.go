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
func tableResultFromDataProxy(result apitypes.DataProxyQueryResult) TableResult {
	columns := append([]string(nil), result.Columns...)
	rows := make([]map[string]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		item := make(map[string]any, len(columns))
		for i, column := range columns {
			if i < len(row) {
				item[column] = row[i]
			} else {
				item[column] = ""
			}
		}
		rows = append(rows, item)
	}
	return TableResult{
		Columns: columns,
		Rows:    rows,
		Metadata: QueryMetadata{
			RowCount:      result.Metadata.RowCount,
			BytesScanned:  result.Metadata.BytesScanned,
			ExecutionTime: result.Metadata.ExecutionTime,
			QueryID:       result.Metadata.QueryID,
			Engine:        result.Metadata.Engine,
			Vendor:        result.Metadata.Vendor,
			Region:        result.Metadata.Region,
			Partial:       result.Metadata.Partial,
			Warnings:      append([]string(nil), result.Metadata.Warnings...),
			Raw:           cloneAnyMap(result.Metadata.Raw),
		},
	}
}
func tableResultFromSchema(result apitypes.DataProxySchemaResult) TableResult {
	rows := make([]map[string]any, 0)
	for _, table := range result.Tables {
		if len(table.Columns) == 0 && len(table.Partitions) == 0 {
			rows = append(rows, map[string]any{
				"database":    table.Database,
				"table":       table.Table,
				"data_source": table.DataSource,
				"location":    table.Location,
			})
			continue
		}
		for _, column := range table.Columns {
			rows = append(rows, map[string]any{
				"database":       table.Database,
				"table":          table.Table,
				"column_name":    column.Name,
				"column_type":    column.Type,
				"column_comment": column.Comment,
				"data_source":    table.DataSource,
				"location":       table.Location,
			})
		}
		for _, partition := range table.Partitions {
			rows = append(rows, map[string]any{
				"database":       table.Database,
				"table":          table.Table,
				"partition_name": partition.Name,
				"partition_type": partition.Type,
				"data_source":    table.DataSource,
				"location":       table.Location,
			})
		}
	}
	return TableResult{
		Columns: []string{
			"database",
			"table",
			"column_name",
			"column_type",
			"column_comment",
			"partition_name",
			"partition_type",
			"data_source",
			"location",
		},
		Rows: rows,
	}
}
func topSQLPlansAsMaps(plans []apitypes.CloudTopSQLPlan) []map[string]any {
	out := make([]map[string]any, 0, len(plans))
	for _, plan := range plans {
		out = append(out, map[string]any{
			"plan_digest":          plan.PlanDigest,
			"plan_text":            plan.PlanText,
			"timestamp_sec":        append([]int64(nil), plan.TimestampSec...),
			"cpu_time_ms":          append([]int64(nil), plan.CPUTimeMS...),
			"exec_count_per_sec":   plan.ExecCountPerSec,
			"duration_per_exec_ms": plan.DurationPerExecMS,
			"scan_records_per_sec": plan.ScanRecordsPerSec,
			"scan_indexes_per_sec": plan.ScanIndexesPerSec,
		})
	}
	return out
}
func tableResultFromTopSQL(items []apitypes.CloudTopSQL) TableResult {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"sql_digest":           item.SQLDigest,
			"sql_text":             item.SQLText,
			"cpu_time_ms":          item.CPUTimeMS,
			"exec_count_per_sec":   item.ExecCountPerSec,
			"duration_per_exec_ms": item.DurationPerExecMS,
			"scan_records_per_sec": item.ScanRecordsPerSec,
			"scan_indexes_per_sec": item.ScanIndexesPerSec,
			"plans": map[string]any{
				"items": topSQLPlansAsMaps(item.Plans),
			},
		})
	}
	return TableResult{
		Columns: []string{
			"sql_digest",
			"sql_text",
			"cpu_time_ms",
			"exec_count_per_sec",
			"duration_per_exec_ms",
			"scan_records_per_sec",
			"scan_indexes_per_sec",
			"plans",
		},
		Rows: rows,
	}
}
func tableResultFromTopSlow(items []apitypes.CloudTopSlowQuery) TableResult {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"db":             item.DB,
			"sql_digest":     item.SQLDigest,
			"sql_text":       item.SQLText,
			"statement_type": item.StatementType,
			"count":          item.Count,
			"sum_latency":    item.SumLatency,
			"max_latency":    item.MaxLatency,
			"avg_latency":    item.AvgLatency,
			"sum_memory":     item.SumMemory,
			"max_memory":     item.MaxMemory,
			"avg_memory":     item.AvgMemory,
			"sum_disk":       item.SumDisk,
			"max_disk":       item.MaxDisk,
			"avg_disk":       item.AvgDisk,
			"detail":         cloneAnyMap(item.Detail),
		})
	}
	return TableResult{
		Columns: []string{
			"db",
			"sql_digest",
			"sql_text",
			"statement_type",
			"count",
			"sum_latency",
			"max_latency",
			"avg_latency",
			"sum_memory",
			"max_memory",
			"avg_memory",
			"sum_disk",
			"max_disk",
			"avg_disk",
			"detail",
		},
		Rows: rows,
	}
}
func listResultFromSlowQuerySamples(items []apitypes.CloudSlowQueryListEntry) ListResult {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"id":            item.ID,
			"item_id":       item.ItemID,
			"source_ref":    item.ItemID,
			"digest":        item.Digest,
			"query":         item.Query,
			"timestamp":     item.Timestamp,
			"query_time":    item.QueryTime,
			"memory_max":    item.MemoryMax,
			"request_count": item.RequestCount,
			"connection_id": item.ConnectionID,
			"raw":           cloneAnyMap(item.Raw),
		})
	}
	return ListResult{Total: len(rows), Items: rows}
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
func objectResultFromProfileDetail(result apitypes.CloudProfileGroupDetail) ObjectResult {
	items := make([]map[string]any, 0, len(result.TargetProfiles))
	for _, item := range result.TargetProfiles {
		items = append(items, map[string]any{
			"profile_type": item.ProfileType,
			"state":        item.State,
			"error":        item.Error,
			"component":    item.Component,
			"address":      item.Address,
			"raw":          cloneAnyMap(item.Raw),
		})
	}
	return ObjectResult{
		Fields: map[string]any{
			"target_profiles": items,
			"raw":             cloneAnyMap(result.Raw),
		},
	}
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
func listResultFromLogSearch(result apitypes.LogSearchResult) ListResult {
	items := make([]map[string]any, 0, len(result.Records))
	for _, record := range result.Records {
		items = append(items, map[string]any{
			"timestamp":  record.Timestamp,
			"component":  record.Component,
			"level":      record.Level,
			"message":    record.Message,
			"item_id":    record.SourceRef,
			"source_ref": record.SourceRef,
		})
	}
	return ListResult{Total: result.Total, Items: items}
}
func tableResultFromSlowQueryRecords(result apitypes.SlowQueryRecordsResult) TableResult {
	rows := make([]map[string]any, 0, len(result.Records))
	for _, record := range result.Records {
		rows = append(rows, map[string]any{
			"digest":      record.Digest,
			"sql_text":    record.SQLText,
			"query_time":  record.QueryTime,
			"exec_count":  record.ExecCount,
			"user":        record.User,
			"db":          record.DB,
			"table_names": append([]string(nil), record.TableNames...),
			"index_names": append([]string(nil), record.IndexNames...),
			"item_id":     record.SourceRef,
			"source_ref":  record.SourceRef,
		})
	}
	return TableResult{
		Columns: []string{
			"digest",
			"sql_text",
			"query_time",
			"exec_count",
			"user",
			"db",
			"table_names",
			"index_names",
			"item_id",
			"source_ref",
		},
		Rows: rows,
		Metadata: QueryMetadata{
			RowCount: int64(result.Total),
		},
	}
}
func tableResultFromConfig(result apitypes.ConfigResult) TableResult {
	rows := make([]map[string]any, 0, len(result.Entries))
	for _, entry := range result.Entries {
		rows = append(rows, map[string]any{
			"component":  entry.Component,
			"key":        entry.Key,
			"value":      entry.Value,
			"item_id":    entry.SourceRef,
			"source_ref": entry.SourceRef,
		})
	}
	return TableResult{
		Columns: []string{"component", "key", "value", "item_id", "source_ref"},
		Rows:    rows,
	}
}
