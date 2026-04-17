package cli

import (
	"bytes"
	"testing"

	clinicapi "github.com/AricSu/tidb-clinic-client"
)

func TestWriteMetricQueryRangeSummaryIncludesRequestAndSampleWindows(t *testing.T) {
	var out bytes.Buffer

	writeMetricQueryRangeSummary(&out, "sum(tidb_server_connections)", 1772776800, 1772780400, "1m", clinicapi.MetricQueryRangeResult{
		Kind: clinicapi.SeriesKindRange,
		Series: []clinicapi.Series{
			{
				Labels: map[string]string{},
				Values: []clinicapi.SeriesPoint{
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
		Kind:      clinicapi.SeriesKindRange,
		IsPartial: true,
		Series: []clinicapi.Series{
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

func TestWriteMetricQueryInstantSummaryIncludesSeriesValues(t *testing.T) {
	var out bytes.Buffer

	writeMetricQueryInstantSummary(&out, "sum(tidb_server_connections)", 1772776903, clinicapi.MetricQueryInstantResult{
		Kind:      clinicapi.SeriesKindInstant,
		IsPartial: true,
		Series: []clinicapi.Series{
			{
				Labels: map[string]string{"instance": "tidb-0"},
				Values: []clinicapi.SeriesPoint{{Timestamp: 1772776903, Value: "2518"}},
			},
		},
	})

	got := out.String()
	want := "" +
		"query=sum(tidb_server_connections)\n" +
		"query_time=1772776903\n" +
		"result_type=vector partial=true series=1\n" +
		"series[0] labels=map[instance:tidb-0] timestamp=1772776903 value=2518\n"
	if got != want {
		t.Fatalf("unexpected output:\n%s\nwant:\n%s", got, want)
	}
}

func TestWriteMetricQuerySeriesSummaryIncludesMatchersAndWindow(t *testing.T) {
	var out bytes.Buffer

	writeMetricQuerySeriesSummary(&out, []string{
		`tidb_server_query_total{instance="tidb-0"}`,
		`tidb_server_query_total{instance="tidb-1"}`,
	}, 1772776800, 1772777400, clinicapi.MetricQuerySeriesResult{
		Kind: clinicapi.SeriesKindSet,
		Series: []clinicapi.Series{
			{Labels: map[string]string{"instance": "tidb-0", "type": "Select"}},
			{Labels: map[string]string{"instance": "tidb-1", "type": "Update"}},
		},
	})

	got := out.String()
	want := "" +
		"match[0]=tidb_server_query_total{instance=\"tidb-0\"}\n" +
		"match[1]=tidb_server_query_total{instance=\"tidb-1\"}\n" +
		"request_window=1772776800..1772777400\n" +
		"series=2\n" +
		"series[0] labels=map[instance:tidb-0 type:Select]\n" +
		"series[1] labels=map[instance:tidb-1 type:Update]\n"
	if got != want {
		t.Fatalf("unexpected output:\n%s\nwant:\n%s", got, want)
	}
}
