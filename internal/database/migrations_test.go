package database_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/woodleighschool/woodstar/internal/database/dbtest"
)

func TestSantaMigrationEnforcesConfigurationAndRuleInvariants(t *testing.T) {
	db, ctx := dbtest.Open(t)

	tx, err := db.Pool().Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var allHostsLabelID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM labels WHERE name = 'All Hosts'`).Scan(&allHostsLabelID); err != nil {
		t.Fatalf("load built-in label: %v", err)
	}

	insertConfiguration := `
		INSERT INTO santa_configurations AS c (
			name, position, client_mode, enable_bundles, enable_transitive_rules,
			enable_all_event_upload, full_sync_interval_seconds, batch_size,
			allowed_path_regex, blocked_path_regex, event_detail_url, event_detail_text
		)
		VALUES ($1, $2, 'monitor', false, false, false, 600, 50, '', '', '', '')
		RETURNING id
	`
	var firstConfigID int64
	if err := tx.QueryRow(ctx, insertConfiguration, "baseline", 0).Scan(&firstConfigID); err != nil {
		t.Fatalf("insert first configuration: %v", err)
	}
	var secondConfigID int64
	if err := tx.QueryRow(ctx, insertConfiguration, "restricted", 1).Scan(&secondConfigID); err != nil {
		t.Fatalf("insert second configuration: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO santa_configuration_labels (label_id, configuration_id)
		VALUES ($1, $2)
	`, allHostsLabelID, firstConfigID); err != nil {
		t.Fatalf("attach first configuration label: %v", err)
	}
	expectPgError(t, tx, "duplicate_configuration_label", "23505", `
		INSERT INTO santa_configuration_labels (label_id, configuration_id)
		VALUES ($1, $2)
	`, allHostsLabelID, secondConfigID)

	expectPgError(t, tx, "remount_without_flags", "23514", `
		INSERT INTO santa_configurations (
			name, position, client_mode, enable_bundles, enable_transitive_rules,
			enable_all_event_upload, full_sync_interval_seconds, batch_size,
			allowed_path_regex, blocked_path_regex, event_detail_url, event_detail_text,
			removable_media_action
		)
		VALUES (
			'invalid-remount', 2, 'monitor', false, false, false, 600, 50, '', '', '', '',
			'remount'
		)
	`)

	var ruleID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO santa_rules (rule_type, identifier)
		VALUES ('binary', 'abc123')
		RETURNING id
	`).Scan(&ruleID); err != nil {
		t.Fatalf("insert rule: %v", err)
	}
	expectPgError(t, tx, "duplicate_rule_identity", "23505", `
		INSERT INTO santa_rules (rule_type, identifier)
		VALUES ('binary', 'abc123')
	`)
	expectPgError(t, tx, "empty_cel_expression", "23514", `
		INSERT INTO santa_rule_includes (rule_id, position, policy, cel_expression)
		VALUES ($1, 0, 'cel', '')
	`, ruleID)
	expectPgError(t, tx, "non_cel_expression", "23514", `
		INSERT INTO santa_rule_includes (rule_id, position, policy, cel_expression)
		VALUES ($1, 0, 'allowlist', 'target.path == "/Applications"')
	`, ruleID)
}

func expectPgError(t *testing.T, tx pgx.Tx, name string, code string, query string, args ...any) {
	t.Helper()

	savepoint := pgx.Identifier{"sp_" + name}.Sanitize()
	if _, err := tx.Exec(t.Context(), "SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("create savepoint: %v", err)
	}

	_, err := tx.Exec(t.Context(), query, args...)
	if !isPgError(err, code) {
		t.Fatalf("%s error = %v, want PostgreSQL error %s", name, err, code)
	}

	if _, err := tx.Exec(t.Context(), "ROLLBACK TO SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("rollback savepoint: %v", err)
	}
	if _, err := tx.Exec(t.Context(), "RELEASE SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("release savepoint: %v", err)
	}
}

func isPgError(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
