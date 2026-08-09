package events

import (
	"fmt"
	"strings"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/listing"
)

var validExecutionDecisions = valueSet(ExecutionDecisionValues)

var validFileAccessDecisions = valueSet(FileAccessDecisionValues)

func valueSet[T comparable](values []T) map[T]struct{} {
	set := make(map[T]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func validateEventInputs(
	executionEvents []ExecutionEventInput,
	fileAccessEvents []FileAccessEventInput,
	standaloneRuleCreationEvents []StandaloneRuleCreationEventInput,
) error {
	for _, event := range executionEvents {
		if event.Decision != ExecutionDecisionBundleBinary && event.OccurredAt.IsZero() {
			return fmt.Errorf("%w: execution event occurred_at is required", fault.ErrInvalidInput)
		}
	}
	for _, event := range fileAccessEvents {
		if event.OccurredAt.IsZero() {
			return fmt.Errorf("%w: file access event occurred_at is required", fault.ErrInvalidInput)
		}
	}
	for _, event := range standaloneRuleCreationEvents {
		if strings.TrimSpace(event.Identifier) == "" {
			return fmt.Errorf("%w: standalone rule identifier is required", fault.ErrInvalidInput)
		}
		if event.Decision == ExecutionDecisionUnknown {
			return fmt.Errorf("%w: standalone rule decision is required", fault.ErrInvalidInput)
		}
		if _, ok := validExecutionDecisions[event.Decision]; !ok {
			return fmt.Errorf("%w: unknown standalone rule decision", fault.ErrInvalidInput)
		}
		if event.OccurredAt.IsZero() {
			return fmt.Errorf("%w: standalone rule occurred_at is required", fault.ErrInvalidInput)
		}
	}
	return nil
}

func validateExecutionEventListParams(params ExecutionEventListParams) error {
	if err := listing.Validate(params.ListParams); err != nil {
		return err
	}
	if params.HostID < 0 {
		return fmt.Errorf("%w: host_id must be non-negative", fault.ErrInvalidInput)
	}
	for _, filter := range params.Decisions {
		if filter == DecisionFilterAllowed || filter == DecisionFilterBlocked {
			continue
		}
		if _, ok := validExecutionDecisions[ExecutionDecision(filter)]; !ok {
			return fmt.Errorf("%w: unknown decision", fault.ErrInvalidInput)
		}
	}
	return nil
}

func validateFileAccessEventListParams(params FileAccessEventListParams) error {
	if err := listing.Validate(params.ListParams); err != nil {
		return err
	}
	if params.HostID < 0 {
		return fmt.Errorf("%w: host_id must be non-negative", fault.ErrInvalidInput)
	}
	for _, decision := range params.Decisions {
		if _, ok := validFileAccessDecisions[decision]; !ok {
			return fmt.Errorf("%w: unknown file access decision", fault.ErrInvalidInput)
		}
	}
	return nil
}
