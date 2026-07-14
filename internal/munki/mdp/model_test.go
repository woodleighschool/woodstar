package mdp

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestDistributionPointMutationValidate(t *testing.T) {
	t.Parallel()
	base := DistributionPointMutation{Name: "Melbourne", ClientBaseURL: "https://mdp.example"}

	cases := []struct {
		name    string
		mutate  func(*DistributionPointMutation)
		wantErr bool
	}{
		{name: "valid", mutate: func(*DistributionPointMutation) {}},
		{name: "blank name", mutate: func(m *DistributionPointMutation) { m.Name = " " }, wantErr: true},
		{
			name:   "valid cidrs",
			mutate: func(m *DistributionPointMutation) { m.ClientCIDRs = []string{"10.0.0.0/8", "::/0"} },
		},
		{
			name:    "bad cidr",
			mutate:  func(m *DistributionPointMutation) { m.ClientCIDRs = []string{"10.0.0.0"} },
			wantErr: true,
		},
		{name: "empty base url allowed", mutate: func(m *DistributionPointMutation) { m.ClientBaseURL = "" }},
		{
			name:    "base url without scheme",
			mutate:  func(m *DistributionPointMutation) { m.ClientBaseURL = "mdp.example" },
			wantErr: true,
		},
		{
			name:    "http base url",
			mutate:  func(m *DistributionPointMutation) { m.ClientBaseURL = "http://mdp.example" },
			wantErr: true,
		},
		{
			name:    "base url with path",
			mutate:  func(m *DistributionPointMutation) { m.ClientBaseURL = "https://mdp.example/packages" },
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mutation := base
			tc.mutate(&mutation)
			err := mutation.validate()
			if tc.wantErr && !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
