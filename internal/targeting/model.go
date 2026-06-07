package targeting

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/humaschema"
)

// Direction describes how a label participates in target resolution.
type Direction string

const (
	// Include labels make a resource eligible for hosts with that label.
	Include Direction = "include"
	// Exclude labels remove a resource from hosts with that label.
	Exclude Direction = "exclude"
)

// DirectionValues lists the supported target directions for schema generation.
var DirectionValues = []Direction{
	Include,
	Exclude,
}

// LabelRef identifies a label used by a target include or exclude set.
type LabelRef struct {
	LabelID int64 `json:"label_id" minimum:"1"`
}

// Result describes the outcome of resolving include/exclude targets for a host.
type Result[T any] struct {
	Matched  bool
	Excluded bool
	Include  T
}

// Schema returns the OpenAPI schema for Direction.
func (Direction) Schema(_ huma.Registry) *huma.Schema {
	return humaschema.StringEnum(DirectionValues...)
}

// ValidDirection reports whether direction is a supported target direction.
func ValidDirection(direction Direction) bool {
	switch direction {
	case Include, Exclude:
		return true
	default:
		return false
	}
}
