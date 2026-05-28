package humaschema

import "github.com/danielgtaylor/huma/v2"

// StringEnum returns a Huma schema for named string enum types.
func StringEnum[T ~string](values ...T) *huma.Schema {
	enum := make([]any, len(values))
	for i, value := range values {
		enum[i] = string(value)
	}
	return &huma.Schema{
		Type: huma.TypeString,
		Enum: enum,
	}
}
