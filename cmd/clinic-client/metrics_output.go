package main

import (
	"fmt"
	"io"

	clinicapi "github.com/AricSu/tidb-clinic-client"
)

func writeMetricQueryRangeSummary(
	out io.Writer,
	query string,
	start, end int64,
	step string,
	result clinicapi.MetricQueryRangeResult,
) {
	fmt.Fprintf(out, "query=%s\n", query)
	fmt.Fprintf(out, "request_window=%d..%d step=%s\n", start, end, step)
	fmt.Fprintf(out, "result_type=%s partial=%t series=%d\n", result.ResultType, result.IsPartial, len(result.Series))
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
