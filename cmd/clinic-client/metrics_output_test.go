package main

import (
	"bytes"
	"testing"

	clinicapi "github.com/AricSu/tidb-clinic-client"
)

func TestWriteMetricQueryRangeSummaryIncludesRequestAndSampleWindows(t *testing.T) {
	var out bytes.Buffer

	writeMetricQueryRangeSummary(&out, "sum(tidb_server_connections)", 1772776800, 1772780400, "1m", clinicapi.MetricQueryRangeResult{
		ResultType: "matrix",
		Series: []clinicapi.MetricSeries{
			{
				Labels: map[string]string{},
				Values: []clinicapi.MetricSample{
					{Timestamp: 1772776860, Value: "1"},
					{Timestamp: 1772780400, Value: "2"},
				},
			},
		},
	})

	got := out.String()
	want := "" +
		"query=sum(tidb_server_connections)\n" +
		"request_window=1772776800..1772780400 step=1m\n" +
		"result_type=matrix partial=false series=1\n" +
		"series[0] labels=map[] samples=2 sample_window=1772776860..1772780400\n" +
		"series[0].sample[0] timestamp=1772776860 value=1\n" +
		"series[0].sample[1] timestamp=1772780400 value=2\n"
	if got != want {
		t.Fatalf("unexpected output:\n%s\nwant:\n%s", got, want)
	}
}

func TestWriteMetricQueryRangeSummaryOmitsSampleWindowWhenSeriesHasNoValues(t *testing.T) {
	var out bytes.Buffer

	writeMetricQueryRangeSummary(&out, "sum(tidb_server_connections)", 1772776800, 1772780400, "1m", clinicapi.MetricQueryRangeResult{
		ResultType: "matrix",
		IsPartial:  true,
		Series: []clinicapi.MetricSeries{
			{
				Labels: map[string]string{"instance": "tidb-0"},
			},
		},
	})

	got := out.String()
	want := "" +
		"query=sum(tidb_server_connections)\n" +
		"request_window=1772776800..1772780400 step=1m\n" +
		"result_type=matrix partial=true series=1\n" +
		"series[0] labels=map[instance:tidb-0] samples=0\n"
	if got != want {
		t.Fatalf("unexpected output:\n%s\nwant:\n%s", got, want)
	}
}
