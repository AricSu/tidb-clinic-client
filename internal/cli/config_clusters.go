package cli

import (
	"strings"
	"time"
)

type cloudEventListConfig struct {
	Base     cliConfig
	Name     string
	Severity *int
}

type eventFlagInputs struct {
	Name        string
	NameSet     bool
	Severity    string
	SeveritySet bool
}

var activeEventFlagInputs eventFlagInputs

func pushEventFlagInputs(next eventFlagInputs) func() {
	previous := activeEventFlagInputs
	activeEventFlagInputs = next
	return func() {
		activeEventFlagInputs = previous
	}
}

func loadCloudEventListConfig(lookup func(string) (string, bool), now func() time.Time) (cloudEventListConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudEventListConfig{}, err
	}
	if activeEventFlagInputs.NameSet || activeEventFlagInputs.SeveritySet {
		severity, err := parseCloudEventSeverity(activeEventFlagInputs.Severity, "--severity")
		if err != nil {
			return cloudEventListConfig{}, err
		}
		return cloudEventListConfig{
			Base:     base,
			Name:     strings.TrimSpace(activeEventFlagInputs.Name),
			Severity: severity,
		}, nil
	}
	name, _ := optionalEnv(lookup, "CLINIC_EVENT_NAME")
	severity, err := optionalCloudEventSeverity(lookup, "CLINIC_EVENT_SEVERITY")
	if err != nil {
		return cloudEventListConfig{}, err
	}
	return cloudEventListConfig{
		Base:     base,
		Name:     name,
		Severity: severity,
	}, nil
}

func optionalCloudEventSeverity(lookup func(string) (string, bool), key string) (*int, error) {
	raw, ok := optionalEnv(lookup, key)
	if !ok {
		return nil, nil
	}
	return parseCloudEventSeverity(raw, key)
}

func parseCloudEventSeverity(raw, key string) (*int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var value int
	switch strings.ToLower(raw) {
	case "info":
		value = 0
	case "warning":
		value = 1
	case "debug":
		value = 2
	case "critical":
		value = 3
	default:
		return nil, &parseEnvError{key: key, message: "must be one of: info, warning, debug, critical"}
	}
	return &value, nil
}
