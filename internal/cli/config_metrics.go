package cli

import (
	"errors"
	"time"
)

func loadMetricsQueryConfig(lookup func(string) (string, bool), now func() time.Time) (cliConfig, error) {
	return loadConfigFromEnv(lookup, now)
}
func loadMetricsSeriesConfig(lookup func(string) (string, bool), now func() time.Time) (cliConfig, error) {
	cfg, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cliConfig{}, err
	}
	if len(cfg.Match) == 0 {
		return cliConfig{}, errors.New("CLINIC_METRICS_MATCH is required for metrics query-series")
	}
	return cfg, nil
}
