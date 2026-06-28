package directory

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodstar/internal/api/schema"
)

// Source identifies the authority that owns a directory object.
type Source string

const (
	SourceLocal Source = "local"
	SourceEntra Source = "entra"
)

var SourceValues = []Source{SourceLocal, SourceEntra}

func (Source) Schema(_ huma.Registry) *huma.Schema {
	return schema.StringEnum(SourceValues...)
}
