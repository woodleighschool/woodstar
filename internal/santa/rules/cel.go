// Package rules validates, targets, and persists Santa rules.
package rules

import (
	"fmt"

	"github.com/google/cel-go/cel"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func validateCELSyntax(expression string) error {
	env, err := cel.NewEnv()
	if err != nil {
		return fmt.Errorf("create cel parser: %w", err)
	}
	_, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("%w: cel_expression is invalid: %s", dbutil.ErrInvalidInput, issues.Err().Error())
	}
	return nil
}
