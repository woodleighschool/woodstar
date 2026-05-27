package queries

import "github.com/woodleighschool/woodstar/internal/scope"

// QueryDefinition is the shared saved-query shape.
type QueryDefinition struct {
	Name        string
	Description string
	Query       string
	LabelScope  scope.LabelScope
}
