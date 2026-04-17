package cli

import (
	"fmt"
	clinicapi "github.com/AricSu/tidb-clinic-client"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func writeMetricQueryRangeSummary(out io.Writer, query string, start, end int64, step string, result clinicapi.MetricQueryRangeResult) {
	fmt.Fprintf(out, "query=%s\n", query)
	fmt.Fprintf(out, "request_window=%d..%d step=%s\n", start, end, step)
	fmt.Fprintf(out, "result_type=%s partial=%t series=%d\n", metricResultType(result.Kind), result.IsPartial, len(result.Series))
	for i, series := range result.Series {
		fmt.Fprintf(out, "series[%d] labels=%v samples=%d", i, series.Labels, len(series.Values))
		if len(series.Values) > 0 {
			fmt.Fprintf(out, " sample_window=%d..%d", series.Values[0].Timestamp, series.Values[len(series.Values)-1].Timestamp)
		}
		fmt.Fprintln(out)
		for j, sample := range series.Values {
			fmt.Fprintf(out, "series[%d].sample[%d] timestamp=%d value=%s\n", i, j, sample.Timestamp, sample.Value)
		}
	}
}
func writeMetricQueryInstantSummary(out io.Writer, query string, at int64, result clinicapi.MetricQueryInstantResult) {
	fmt.Fprintf(out, "query=%s\n", query)
	fmt.Fprintf(out, "query_time=%d\n", at)
	fmt.Fprintf(out, "result_type=%s partial=%t series=%d\n", metricResultType(result.Kind), result.IsPartial, len(result.Series))
	for i, series := range result.Series {
		timestamp, value := int64(0), ""
		if len(series.Values) > 0 {
			timestamp, value = series.Values[0].Timestamp, series.Values[0].Value
		}
		fmt.Fprintf(out, "series[%d] labels=%v timestamp=%d value=%s\n", i, series.Labels, timestamp, value)
	}
}
func writeMetricQuerySeriesSummary(out io.Writer, match []string, start, end int64, result clinicapi.MetricQuerySeriesResult) {
	for i, item := range match {
		fmt.Fprintf(out, "match[%d]=%s\n", i, item)
	}
	if start > 0 || end > 0 {
		fmt.Fprintf(out, "request_window=%d..%d\n", start, end)
	}
	fmt.Fprintf(out, "series=%d\n", len(result.Series))
	for i, series := range result.Series {
		fmt.Fprintf(out, "series[%d] labels=%v\n", i, series.Labels)
	}
}
func outputPathOrDefault(path, fallback string) string {
	if strings.TrimSpace(path) != "" {
		return strings.TrimSpace(path)
	}
	return fallback
}
func writeArtifact(out io.Writer, outputPath string, artifact clinicapi.DownloadedArtifact) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, artifact.Bytes, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(out, "wrote=%s size=%d\n", outputPath, len(artifact.Bytes))
	return nil
}
func metricResultType(kind clinicapi.SeriesKind) string {
	switch kind {
	case clinicapi.SeriesKindRange:
		return "matrix"
	case clinicapi.SeriesKindInstant:
		return "vector"
	case clinicapi.SeriesKindSet:
		return "series"
	default:
		return string(kind)
	}
}
