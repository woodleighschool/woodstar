package syncstate

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestCursorRoundTrip(t *testing.T) {
	t.Parallel()

	cursor, err := encodeCursor(42)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	offset, err := decodeCursor(cursor)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if offset != 42 {
		t.Fatalf("offset = %d, want 42", offset)
	}
}

func TestDecodeCursorRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	for _, cursor := range []string{"not-base64", "bm90LWpzb24", "eyJvZmZzZXQiOi0xfQ"} {
		if _, err := decodeCursor(cursor); !errors.Is(err, dbutil.ErrInvalidInput) {
			t.Errorf("decodeCursor(%q) error = %v, want ErrInvalidInput", cursor, err)
		}
	}
}
