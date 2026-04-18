package cli

import "time"

type cloudProfilingDownloadConfig struct {
	Base        cliConfig
	Timestamp   int64
	ProfileType string
	Component   string
	Address     string
	DataFormat  string
	OutputPath  string
}

func loadCloudProfilingDownloadConfig(lookup func(string) (string, bool), now func() time.Time) (cloudProfilingDownloadConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudProfilingDownloadConfig{}, err
	}
	tsRaw, err := requiredEnv(lookup, "CLINIC_PROFILE_TS")
	if err != nil {
		return cloudProfilingDownloadConfig{}, err
	}
	ts, err := parseInt64Env("CLINIC_PROFILE_TS", tsRaw)
	if err != nil {
		return cloudProfilingDownloadConfig{}, err
	}
	profileType, err := requiredEnv(lookup, "CLINIC_PROFILE_TYPE")
	if err != nil {
		return cloudProfilingDownloadConfig{}, err
	}
	component, err := requiredEnv(lookup, "CLINIC_PROFILE_COMPONENT")
	if err != nil {
		return cloudProfilingDownloadConfig{}, err
	}
	address, err := requiredEnv(lookup, "CLINIC_PROFILE_ADDRESS")
	if err != nil {
		return cloudProfilingDownloadConfig{}, err
	}
	dataFormat, _ := optionalEnv(lookup, "CLINIC_PROFILE_DATA_FORMAT")
	outputPath, _ := optionalEnv(lookup, "CLINIC_OUTPUT_PATH")
	return cloudProfilingDownloadConfig{
		Base:        base,
		Timestamp:   ts,
		ProfileType: profileType,
		Component:   component,
		Address:     address,
		DataFormat:  dataFormat,
		OutputPath:  outputPath,
	}, nil
}
