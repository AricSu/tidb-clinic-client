package cli

import "time"

func loadMetricsQueryConfig(lookup func(string) (string, bool), now func() time.Time) (cliConfig, error) {
	return loadConfigFromEnv(lookup, now)
}

func loadMetricsCompileConfig(lookup func(string) (string, bool), now func() time.Time) (cliConfig, error) {
	return loadConfigFromEnv(lookup, now)
}
