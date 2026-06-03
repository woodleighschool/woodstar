package database

import (
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSQLState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "foreign key", err: pgError(pgerrcode.ForeignKeyViolation), want: pgerrcode.ForeignKeyViolation},
		{name: "unique", err: pgError(pgerrcode.UniqueViolation), want: pgerrcode.UniqueViolation},
		{name: "plain error", err: errors.New("plain"), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SQLState(tt.err); got != tt.want {
				t.Fatalf("SQLState = %q, want %q", got, tt.want)
			}
		})
	}
}

func pgError(code string) error {
	return &pgconn.PgError{Code: code}
}
