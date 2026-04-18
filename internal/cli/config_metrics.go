package cli

import "time"

type metricsCompileFlagInputs struct {
	ExprDescription    string
	ExprDescriptionSet bool
}

var activeMetricsCompileFlagInputs metricsCompileFlagInputs

func pushMetricsCompileFlagInputs(next metricsCompileFlagInputs) func() {
	previous := activeMetricsCompileFlagInputs
	activeMetricsCompileFlagInputs = next
	return func() {
		activeMetricsCompileFlagInputs = previous
	}
}

func loadMetricsQueryConfig(lookup func(string) (string, bool), now func() time.Time) (cliConfig, error) {
	return loadConfigFromEnv(lookup, now)
}

func loadMetricsCompileConfig(lookup func(string) (string, bool), now func() time.Time) (cliConfig, error) {
	cfg, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cliConfig{}, err
	}
	if activeMetricsCompileFlagInputs.ExprDescriptionSet {
		cfg.ExprDescription = activeMetricsCompileFlagInputs.ExprDescription
	}
	return cfg, nil
}
