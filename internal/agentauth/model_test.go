package agentauth

import (
	"errors"
	"testing"
)

func TestAgentSecretCreateValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   AgentSecretCreate
		want error
	}{
		{
			name: "valid",
			in:   AgentSecretCreate{Agent: AgentOrbit, Value: "orbit-secret-value-long-enough-32"},
		},
		{
			name: "unknown agent",
			in:   AgentSecretCreate{Agent: Agent("mdm"), Value: "mdm-secret-value-long-enough-32"},
			want: ErrInvalidAgent,
		},
		{
			name: "short secret",
			in:   AgentSecretCreate{Agent: AgentOrbit, Value: "short"},
			want: ErrInvalidSecret,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.in
			got.normalize()
			if err := got.validate(); !errors.Is(err, tt.want) {
				t.Fatalf("validate error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestAgentSecretMutationRejectsShortSecret(t *testing.T) {
	t.Parallel()

	err := (AgentSecretMutation{Value: "short"}).validate()
	if !errors.Is(err, ErrInvalidSecret) {
		t.Fatalf("validate error = %v, want ErrInvalidSecret", err)
	}
}
