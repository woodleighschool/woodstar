package destination

import (
	"context"
	"encoding/json"

	"github.com/invopop/jsonschema"
	"github.com/woodleighschool/stemma/plugin"
)

// Register exposes the transport through Stemma's shared operation registry.
func Register(registry *plugin.Registry) error {
	reflector := jsonschema.Reflector{DoNotReference: true}
	return registry.Register(plugin.Operation{
		Name: "woodstar.munki", Kind: "reconcile", SideEffects: "remote", Methods: []string{"validate", "plan", "apply"},
		RequiresInspection: true,
		Content:            &plugin.ContentContract{Formats: []string{"pkg", "dmg"}, SourceFree: true},
		ConfigSchema:       json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","pattern":"^https://"},"api_key":{"type":"string","minLength":1,"writeOnly":true},"ca_file":{"type":"string"}},"required":["url","api_key"],"additionalProperties":false}`),
		MetadataSchema:     raw(metadataSchema()),
		InputSchema:        raw(reflector.Reflect(plugin.ReconcileRequest{})),
		OutputSchema:       raw(reflector.Reflect(plugin.ReconcileResponse{})),
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
