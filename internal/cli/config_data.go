package cli

import "time"

type collectedDataDownloadConfig struct {
	Base       cliConfig
	OutputPath string
}

func loadCollectedDataDownloadConfig(lookup func(string) (string, bool), now func() time.Time) (collectedDataDownloadConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return collectedDataDownloadConfig{}, err
	}
	outputPath, _ := optionalEnv(lookup, "CLINIC_OUTPUT_PATH")
	return collectedDataDownloadConfig{
		Base:       base,
		OutputPath: outputPath,
	}, nil
}
