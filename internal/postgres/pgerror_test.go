package postgres_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/woodleighschool/woodstar/internal/fault"
	"github.com/woodleighschool/woodstar/internal/postgres"
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
		{"no rows", pgx.ErrNoRows, fault.ErrNotFound},
		{"foreign key", pgErr(pgerrcode.ForeignKeyViolation), fault.ErrNotFound},
		{"unique", pgErr(pgerrcode.UniqueViolation), fault.ErrAlreadyExists},
		{"check", pgErr(pgerrcode.CheckViolation), fault.ErrInvalidInput},
		{"not null", pgErr(pgerrcode.NotNullViolation), fault.ErrInvalidInput},
		{"invalid text", pgErr(pgerrcode.InvalidTextRepresentation), fault.ErrInvalidInput},
		{"unmapped passes through", other, other},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := postgres.MutationError(tt.in); !errors.Is(got, tt.want) {
				t.Fatalf("MutationError(%v) = %v, want errors.Is %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestGetError(t *testing.T) {
	t.Parallel()
	other := errors.New("boom")

	if got := postgres.GetError(pgx.ErrNoRows); !errors.Is(got, fault.ErrNotFound) {
		t.Fatalf("GetError(pgx.ErrNoRows) = %v, want errors.Is fault.ErrNotFound", got)
	}
	if got := postgres.GetError(other); !errors.Is(got, other) {
		t.Fatalf("GetError(other) = %v, want original error", got)
	}
}

func TestDeleteConflict(t *testing.T) {
	t.Parallel()
	t.Run("references become conflict with message", func(t *testing.T) {
		t.Parallel()
		for _, code := range []string{pgerrcode.ForeignKeyViolation, pgerrcode.RestrictViolation} {
			got := postgres.DeleteConflict(pgErr(code), "widget is still referenced")
			if !errors.Is(got, fault.ErrConflict) {
				t.Fatalf("DeleteConflict(%s) = %v, want errors.Is fault.ErrConflict", code, got)
			}
			if !strings.Contains(got.Error(), "widget is still referenced") {
				t.Fatalf("DeleteConflict(%s) message = %q, want the supplied message", code, got)
			}
		}
	})

	t.Run("falls back to MutationError", func(t *testing.T) {
		t.Parallel()
		got := postgres.DeleteConflict(pgErr(pgerrcode.UniqueViolation), "unused")
		if !errors.Is(got, fault.ErrAlreadyExists) {
			t.Fatalf("DeleteConflict unique = %v, want errors.Is fault.ErrAlreadyExists", got)
		}
	})
}
