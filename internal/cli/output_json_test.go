package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	clinicapi "github.com/AricSu/tidb-clinic-client"
)

func TestWriteJSONUsesStructuredLowercaseFields(t *testing.T) {
	var out bytes.Buffer

	err := writeJSON(&out, clinicapi.MetricQueryRangeResult{
		Kind:      clinicapi.SeriesKindRange,
		IsPartial: true,
		Series: []clinicapi.Series{
			{
				Labels: map[string]string{"instance": "tidb-0"},
				Values: []clinicapi.SeriesPoint{{Timestamp: 1772776860, Value: "1"}},
			},
		},
		Metadata: clinicapi.QueryMetadata{
			RowCount: 1,
			Warnings: []string{"partial"},
		},
	})
	if err != nil {
		t.Fatalf("writeJSON failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if _, ok := payload["kind"]; !ok {
		t.Fatalf("expected lowercase kind field, got=%s", out.String())
	}
	if _, ok := payload["Kind"]; ok {
		t.Fatalf("expected uppercase Kind field to be absent, got=%s", out.String())
	}
	if got, ok := payload["isPartial"].(bool); !ok || !got {
		t.Fatalf("expected isPartial=true, got=%v", payload["isPartial"])
	}
}

func TestWriteArtifactPrintsJSONMetadata(t *testing.T) {
	var out bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "bundle.zip")

	err := writeArtifact(&out, outputPath, clinicapi.DownloadedArtifact{
		Filename:    "bundle.zip",
		ContentType: "application/octet-stream",
		Bytes:       []byte("bundle-content"),
	})
	if err != nil {
		t.Fatalf("writeArtifact failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if got := payload["path"]; got != outputPath {
		t.Fatalf("expected path %q, got %#v", outputPath, got)
	}
	if got := payload["filename"]; got != "bundle.zip" {
		t.Fatalf("expected filename, got %#v", got)
	}
	if got := int(payload["size"].(float64)); got != len("bundle-content") {
		t.Fatalf("expected size %d, got %d", len("bundle-content"), got)
	}
}
