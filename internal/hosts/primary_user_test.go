package hosts

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestValidatePrimaryUser(t *testing.T) {
	t.Parallel()

	email, err := validatePrimaryUser("  student@example.test  ", PrimaryUserSourceOrbitProfile)
	if err != nil {
		t.Fatalf("validate primary user: %v", err)
	}
	if email != "student@example.test" {
		t.Fatalf("normalized email = %q, want student@example.test", email)
	}

	for name, input := range map[string]struct {
		email  string
		source PrimaryUserSource
	}{
		"blank email":     {source: PrimaryUserSourceOrbitProfile},
		"malformed email": {email: "not-an-email", source: PrimaryUserSourceOrbitProfile},
		"unknown source":  {email: "student@example.test", source: PrimaryUserSource("directory")},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := validatePrimaryUser(input.email, input.source); !errors.Is(err, dbutil.ErrInvalidInput) {
				t.Fatalf("validate primary user error = %v, want ErrInvalidInput", err)
			}
		})
	}
}
