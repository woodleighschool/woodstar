package events

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestValidateEventInputsRequiresOccurrenceTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		execution  []ExecutionEventInput
		fileAccess []FileAccessEventInput
		standalone []StandaloneRuleCreationEventInput
	}{
		{
			name: "execution event",
			execution: []ExecutionEventInput{{
				FileSHA256: "sha-without-time",
				Decision:   ExecutionDecisionBlockBinary,
			}},
		},
		{
			name:       "file access event",
			fileAccess: []FileAccessEventInput{{Decision: FileAccessDecisionDenied}},
		},
		{
			name: "standalone rule event",
			standalone: []StandaloneRuleCreationEventInput{{
				Identifier: "identifier",
				Decision:   ExecutionDecisionAllowBinary,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := validateEventInputs(tt.execution, tt.fileAccess, tt.standalone); !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("validateEventInputs error = %v, want ErrInvalidInput", err)
			}
		})
	}
}
