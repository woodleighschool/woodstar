package dbutil_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func pgErr(code string) error {
	return &pgconn.PgError{Code: code}
}

func TestMutationError(t *testing.T) {
	t.Parallel()
	other := errors.New("boom")
	tests := []struct {
		name string
		in   error
		want error
	}{
		{"no rows", pgx.ErrNoRows, dbutil.ErrNotFound},
		{"foreign key", pgErr(pgerrcode.ForeignKeyViolation), dbutil.ErrNotFound},
		{"unique", pgErr(pgerrcode.UniqueViolation), dbutil.ErrAlreadyExists},
		{"check", pgErr(pgerrcode.CheckViolation), dbutil.ErrInvalidInput},
		{"not null", pgErr(pgerrcode.NotNullViolation), dbutil.ErrInvalidInput},
		{"invalid text", pgErr(pgerrcode.InvalidTextRepresentation), dbutil.ErrInvalidInput},
		{"unmapped passes through", other, other},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := dbutil.MutationError(tt.in); !errors.Is(got, tt.want) {
				t.Fatalf("MutationError(%v) = %v, want errors.Is %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestGetError(t *testing.T) {
	t.Parallel()
	other := errors.New("boom")

	if got := dbutil.GetError(pgx.ErrNoRows); !errors.Is(got, dbutil.ErrNotFound) {
		t.Fatalf("GetError(pgx.ErrNoRows) = %v, want errors.Is ErrNotFound", got)
	}
	if got := dbutil.GetError(other); got != other {
		t.Fatalf("GetError(other) = %v, want original error", got)
	}
}

func TestDeleteConflict(t *testing.T) {
	t.Parallel()
	t.Run("references become conflict with message", func(t *testing.T) {
		t.Parallel()
		for _, code := range []string{pgerrcode.ForeignKeyViolation, pgerrcode.RestrictViolation} {
			got := dbutil.DeleteConflict(pgErr(code), "widget is still referenced")
			if !errors.Is(got, dbutil.ErrConflict) {
				t.Fatalf("DeleteConflict(%s) = %v, want errors.Is ErrConflict", code, got)
			}
			if !strings.Contains(got.Error(), "widget is still referenced") {
				t.Fatalf("DeleteConflict(%s) message = %q, want the supplied message", code, got)
			}
		}
	})

	t.Run("falls back to MutationError", func(t *testing.T) {
		t.Parallel()
		got := dbutil.DeleteConflict(pgErr(pgerrcode.UniqueViolation), "unused")
		if !errors.Is(got, dbutil.ErrAlreadyExists) {
			t.Fatalf("DeleteConflict unique = %v, want errors.Is ErrAlreadyExists", got)
		}
	})
}
