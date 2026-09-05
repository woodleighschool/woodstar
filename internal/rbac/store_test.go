package rbac

import (
	"errors"
	"testing"

	"github.com/woodleighschool/goodies/auth/authz"
)

func TestDatabaseAccessRejectsUnexpectedLevels(t *testing.T) {
	for _, level := range []int16{-1, 0, 3, 32767} {
		if _, err := accessFromLevel(level); !errors.Is(err, authz.ErrInvalidAccess) {
			t.Fatalf("database level %d: %v", level, err)
		}
	}
}
