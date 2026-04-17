package clinicapi

import (
	"encoding/json"
	"testing"
)

func TestDecodeMetricInstantSeriesSupportsEmptyMatrix(t *testing.T) {
	series, err := decodeMetricInstantSeries("matrix", json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("decodeMetricInstantSeries returned error: %v", err)
	}
	if len(series) != 0 {
		t.Fatalf("expected no series, got %+v", series)
	}
}

func TestDecodeMetricInstantSeriesSupportsMatrixRows(t *testing.T) {
	series, err := decodeMetricInstantSeries("matrix", json.RawMessage(`[
		{
			"metric": {"instance":"tidb-0"},
			"values": [[1772780340, "1"], [1772780400, "2"]]
		}
	]`))
	if err != nil {
		t.Fatalf("decodeMetricInstantSeries returned error: %v", err)
	}
	if len(series) != 1 {
		t.Fatalf("expected one series, got %+v", series)
	}
	if len(series[0].Values) != 1 || series[0].Values[0].Timestamp != 1772780400 || series[0].Values[0].Value != "2" {
		t.Fatalf("unexpected sample: %+v", series[0].Values)
	}
	if series[0].Labels["instance"] != "tidb-0" {
		t.Fatalf("unexpected labels: %+v", series[0].Labels)
	}
}
