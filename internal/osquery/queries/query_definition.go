package queries

import (
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/dbutil"
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

func CleanQueryDefinition(params QueryDefinition) (QueryDefinition, error) {
	params.Name = strings.TrimSpace(params.Name)
	params.Description = strings.TrimSpace(params.Description)
	params.Query = strings.TrimSpace(params.Query)
	targets, err := platforms.CleanTargets(params.Platforms)
	if err != nil {
		return QueryDefinition{}, fmt.Errorf("%w: %w", dbutil.ErrInvalidInput, err)
	}
	params.Platforms = targets
	params.LabelScope = scope.NormalizeLabelScope(params.LabelScope)
	if params.Name == "" {
		return QueryDefinition{}, fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if params.Query == "" {
		return QueryDefinition{}, fmt.Errorf("%w: query is required", dbutil.ErrInvalidInput)
	}
	return params, nil
}
