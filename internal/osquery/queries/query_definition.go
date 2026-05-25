package queries

import (
	"github.com/woodleighschool/woodstar/internal/platforms"
	"github.com/woodleighschool/woodstar/internal/scope"
)

// QueryDefinition is the shared editable shape for saved osquery definitions.
type QueryDefinition struct {
	Name        string
	Description string
	Query       string
	Platforms   []platforms.Platform
	LabelScope  scope.LabelScope
}
