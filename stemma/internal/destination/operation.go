package destination

import (
	"context"
	"encoding/json"

	"github.com/woodleighschool/stemma/plugin"
)

// Register exposes the transport through Stemma's shared operation registry.
func Register(registry *plugin.Registry) error {
	return registry.Register(plugin.Operation{
		Name: "woodstar.munki", Kind: "reconcile", SideEffects: "remote", Methods: []string{"validate", "plan", "apply"},
		ConfigSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","pattern":"^https://"},"api_key":{"type":"string","minLength":1},"ca_file":{"type":"string"}},"required":["url","api_key"],"additionalProperties":false}`),
		InputSchema:  json.RawMessage(`{"type":"object","properties":{"method":{"type":"string"},"identity":{"type":"object"},"config":{"type":"object"},"metadata":{"type":"object","properties":{"targets":{"type":"object"}},"additionalProperties":false},"binding":true,"artifact":{"type":"object"},"inputs":{"type":"object","properties":{"installer":{"type":"object"}},"additionalProperties":false},"facts":{"type":"object"}},"required":["config","artifact"],"additionalProperties":false}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"changes":{"type":"array"},"binding":true},"additionalProperties":false}`),
	}, func(ctx context.Context, envelope plugin.Request) (plugin.Response, error) {
		var request plugin.ReconcileRequest
		if err := decode(envelope.Input, &request); err != nil {
			return plugin.Response{}, err
		}
		request.Method = envelope.Method
		result, err := Handle(ctx, request)
		return plugin.Response{Output: raw(result)}, err
	})
}
