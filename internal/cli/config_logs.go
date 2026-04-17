package cli

import "time"

type cloudLokiQueryConfig struct {
	Base      cliConfig
	Query     string
	Limit     int
	Direction string
}
type cloudLokiLabelValuesConfig struct {
	Base      cliConfig
	LabelName string
}

func loadCloudLokiQueryConfig(lookup func(string) (string, bool), now func() time.Time) (cloudLokiQueryConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudLokiQueryConfig{}, err
	}
	query, err := requiredEnv(lookup, "CLINIC_LOKI_QUERY")
	if err != nil {
		return cloudLokiQueryConfig{}, err
	}
	limit, err := optionalPositiveIntEnv(lookup, "CLINIC_LOKI_LIMIT")
	if err != nil {
		return cloudLokiQueryConfig{}, err
	}
	direction, _ := optionalEnv(lookup, "CLINIC_LOKI_DIRECTION")
	return cloudLokiQueryConfig{
		Base:      base,
		Query:     query,
		Limit:     limit,
		Direction: direction,
	}, nil
}
func loadCloudLokiLabelValuesConfig(lookup func(string) (string, bool), now func() time.Time) (cloudLokiLabelValuesConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudLokiLabelValuesConfig{}, err
	}
	labelName, err := requiredEnv(lookup, "CLINIC_LOKI_LABEL")
	if err != nil {
		return cloudLokiLabelValuesConfig{}, err
	}
	return cloudLokiLabelValuesConfig{Base: base, LabelName: labelName}, nil
}
