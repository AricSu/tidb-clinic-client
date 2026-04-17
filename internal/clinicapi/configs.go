package clinicapi

import (
	"context"
	"net/url"
	"strings"
)

func (c *configClient) Get(ctx context.Context, req ConfigRequest) (ConfigResult, error) {
	if c == nil || c.transport == nil {
		return ConfigResult{}, &Error{Class: ErrBackend, Message: "config client is nil"}
	}
	route, err := routeFromItemContext(configEndpoint, req.Context, req.ItemID)
	if err != nil {
		return ConfigResult{}, err
	}
	query := url.Values{}
	query.Set("itemID", strings.TrimSpace(req.ItemID))
	var raw any
	if err := c.transport.getJSON(ctx, configEndpoint, query, route.headers, route.trace, &raw); err != nil {
		return ConfigResult{}, err
	}
	return normalizeConfigResult(raw, req.ItemID), nil
}
func normalizeConfigResult(raw any, itemID string) ConfigResult {
	_, entries := unwrapCollection(raw)
	out := ConfigResult{Entries: make([]ConfigEntry, 0, len(entries))}
	for _, row := range entries {
		out.Entries = append(out.Entries, ConfigEntry{
			Component: asTrimmedString(firstPresent(row, "component", "module")),
			Key:       asTrimmedString(firstPresent(row, "key", "name")),
			Value:     asTrimmedString(firstPresent(row, "value", "val")),
			SourceRef: strings.TrimSpace(itemID),
		})
	}
	return out
}
