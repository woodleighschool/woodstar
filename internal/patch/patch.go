// Package patch applies typed JSON merge patches to editable resource models.
package patch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/fault"
)

// Document contains a validated sparse mutation. Objects merge recursively,
// arrays replace, and null clears optional strings and pointers. Decode with
// [json.Unmarshal]; the zero value is an empty patch. Domain validation belongs
// to the caller after Apply and normalization.
type Document[T any] struct {
	raw json.RawMessage
}

// UnmarshalJSON rejects unknown fields, invalid types, and non-clearable nulls.
func (p *Document[T]) UnmarshalJSON(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("%w: patch: %w", fault.ErrInvalidInput, err)
	}
	registry := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	contract := p.Schema(registry)
	if err := checkFields(contract, value); err != nil {
		return fmt.Errorf("%w: patch: %w", fault.ErrInvalidInput, err)
	}
	result := &huma.ValidateResult{}
	huma.Validate(registry, contract, &huma.PathBuffer{}, huma.ModeWriteToServer, value, result)
	if len(result.Errors) != 0 {
		return fmt.Errorf("%w: patch: %w", fault.ErrInvalidInput, result.Errors[0])
	}
	// Decode the original bytes too, preserving integer precision and checking
	// Go-specific representations such as timestamps and bounded integers.
	var typed T
	if err := json.Unmarshal(data, &typed); err != nil {
		return fmt.Errorf("%w: patch: %w", fault.ErrInvalidInput, err)
	}
	p.raw = bytes.Clone(bytes.TrimSpace(data))
	return nil
}

// Huma accepts case-insensitive keys and null for omitted/optional fields.
// Merge patches need exact keys and explicit nullability even on optional fields.
func checkFields(contract *huma.Schema, value any) error {
	if value == nil && !contract.Nullable {
		return fmt.Errorf("null is not allowed")
	}
	switch value := value.(type) {
	case map[string]any:
		for key, item := range value {
			field, ok := contract.Properties[key]
			if !ok {
				return fmt.Errorf("unknown field %q", key)
			}
			if err := checkFields(field, item); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
	case []any:
		if contract.Items != nil {
			for i, item := range value {
				if err := checkFields(contract.Items, item); err != nil {
					return fmt.Errorf("[%d]: %w", i, err)
				}
			}
		}
	}
	return nil
}

// Bytes returns an independent copy of the sparse JSON document.
func (p Document[T]) Bytes() json.RawMessage {
	if len(p.raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return bytes.Clone(p.raw)
}

// MarshalJSON preserves the supplied fields, including explicit nulls.
func (p Document[T]) MarshalJSON() ([]byte, error) {
	return p.Bytes(), nil
}

// Apply merges into a copy of current without changing its slices or pointers.
func (p Document[T]) Apply(current T) (T, error) {
	var result T
	data, err := json.Marshal(current)
	if err != nil {
		return result, fmt.Errorf("patch: encode current mutation: %w", err)
	}
	merged, err := merge(data, p.Bytes())
	if err != nil {
		return result, fmt.Errorf("patch: merge mutation: %w", err)
	}
	if err := json.Unmarshal(merged, &result); err != nil {
		return result, fmt.Errorf("patch: decode merged mutation: %w", err)
	}
	return result, nil
}

func merge(current, document []byte) ([]byte, error) {
	var fields map[string]json.RawMessage
	if len(document) == 0 || document[0] != '{' {
		return document, nil
	}
	if err := json.Unmarshal(document, &fields); err != nil {
		return nil, err
	}
	var target map[string]json.RawMessage
	if len(current) > 0 && current[0] == '{' {
		if err := json.Unmarshal(current, &target); err != nil {
			return nil, err
		}
	}
	if target == nil {
		target = make(map[string]json.RawMessage)
	}
	for key, value := range fields {
		if bytes.Equal(value, []byte("null")) {
			delete(target, key)
			continue
		}
		merged, err := merge(target[key], value)
		if err != nil {
			return nil, err
		}
		target[key] = merged
	}
	return json.Marshal(target)
}

// Schema derives the patch contract from the mutation's existing field schemas.
// Array elements remain complete objects; only merged objects become partial.
func (Document[T]) Schema(registry huma.Registry) *huma.Schema {
	t := reflect.TypeFor[T]()
	return schema(registry, registry.Schema(t, true, ""), t, true, false)
}

func schema(registry huma.Registry, source *huma.Schema, t reflect.Type, partial, nullable bool) *huma.Schema {
	for source.Ref != "" {
		source = registry.SchemaFromRef(source.Ref)
	}
	result := *source
	result.Nullable = nullable
	if nullable && len(result.Enum) > 0 {
		result.Enum = append(slices.Clone(result.Enum), nil)
	}
	result.Default = nil
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch result.Type {
	case huma.TypeObject:
		result.Properties = make(map[string]*huma.Schema, len(source.Properties))
		for _, field := range reflect.VisibleFields(t) {
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			property, ok := source.Properties[name]
			if !ok {
				continue
			}
			clearable := field.Type.Kind() == reflect.Pointer ||
				field.Type.Kind() == reflect.String && property.MinLength == nil &&
					!strings.Contains(field.Tag.Get("validate"), "required") && field.Tag.Get("nullable") != "false"
			result.Properties[name] = schema(registry, property, field.Type, partial, clearable)
		}
		if partial {
			result.Required = nil
		}
	case huma.TypeArray:
		result.Items = schema(registry, source.Items, t.Elem(), false, t.Elem().Kind() == reflect.Pointer)
	}
	result.PrecomputeMessages()
	return &result
}
