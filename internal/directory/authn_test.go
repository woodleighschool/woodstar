package directory

import (
	"errors"
	"fmt"
	"testing"

	"github.com/woodleighschool/goodies/auth/authn"

	"github.com/woodleighschool/woodstar/internal/fault"
)

func TestAuthnStoreErrorMapping(t *testing.T) {
	broken := errors.New("store unavailable")
	for _, tc := range []struct{ input, want error }{
		{nil, nil},
		{fmt.Errorf("lookup: %w", fault.ErrNotFound), authn.ErrPrincipalNotFound},
		{fmt.Errorf("lookup: %w", broken), broken},
	} {
		if err := authnStoreError(tc.input); !errors.Is(err, tc.want) {
			t.Fatalf("mapped error = %v, want %v", err, tc.want)
		}
	}
}
