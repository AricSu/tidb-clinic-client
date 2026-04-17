package cli

import "time"

type cloudDiagnosticDownloadConfig struct {
	Base       cliConfig
	Key        string
	OutputPath string
}

func loadCloudDiagnosticDownloadConfig(lookup func(string) (string, bool), now func() time.Time) (cloudDiagnosticDownloadConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudDiagnosticDownloadConfig{}, err
	}
	key, err := requiredEnv(lookup, "CLINIC_DIAGNOSTIC_KEY")
	if err != nil {
		return cloudDiagnosticDownloadConfig{}, err
	}
	outputPath, _ := optionalEnv(lookup, "CLINIC_OUTPUT_PATH")
	return cloudDiagnosticDownloadConfig{Base: base, Key: key, OutputPath: outputPath}, nil
}
