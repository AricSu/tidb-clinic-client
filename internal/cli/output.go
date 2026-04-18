package cli

import (
	"encoding/json"
	"fmt"
	clinicapi "github.com/AricSu/tidb-clinic-client"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func writeJSON(out io.Writer, value any) error {
	body, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(body))
	return err
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
	return writeJSON(out, struct {
		Path        string `json:"path"`
		Size        int    `json:"size"`
		Filename    string `json:"filename,omitempty"`
		ContentType string `json:"contentType,omitempty"`
	}{
		Path:        outputPath,
		Size:        len(artifact.Bytes),
		Filename:    strings.TrimSpace(artifact.Filename),
		ContentType: strings.TrimSpace(artifact.ContentType),
	})
}
