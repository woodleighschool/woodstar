package database_test

import (
	"context"
	"testing"

	"github.com/jackc/pgerrcode"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/dbtest"
)

func TestTargetSchemaRejectsDuplicateLabelAcrossDirections(t *testing.T) {
	db, ctx := dbtest.Open(t)

	labelID := createTargetSchemaLabel(t, ctx, db, "Duplicate Direction Label")
	checkID := createTargetSchemaCheck(t, ctx, db)
	mustExecTargetSchema(t, ctx, db, `
		INSERT INTO osquery_check_targets (check_id, direction, position, label_id)
		VALUES ($1, 'include', 0, $2)
	`, checkID, labelID)

	_, err := db.Pool().Exec(ctx, `
		INSERT INTO osquery_check_targets (check_id, direction, position, label_id)
		VALUES ($1, 'exclude', 0, $2)
	`, checkID, labelID)
	expectTargetSchemaSQLState(t, err, pgerrcode.UniqueViolation)
}

func TestTargetSchemaPersistsIncludeOrderByPosition(t *testing.T) {
	db, ctx := dbtest.Open(t)

	firstLabelID := createTargetSchemaLabel(t, ctx, db, "First Ordered Label")
	secondLabelID := createTargetSchemaLabel(t, ctx, db, "Second Ordered Label")
	reportID := createTargetSchemaReport(t, ctx, db)
	mustExecTargetSchema(t, ctx, db, `
		INSERT INTO osquery_report_targets (report_id, direction, position, label_id)
		VALUES
			($1, 'include', 1, $2),
			($1, 'include', 0, $3)
	`, reportID, firstLabelID, secondLabelID)

	rows, err := db.Pool().Query(ctx, `
		SELECT label_id
		FROM osquery_report_targets
		WHERE report_id = $1 AND direction = 'include'
		ORDER BY position
	`, reportID)
	if err != nil {
		t.Fatalf("query report target order: %v", err)
	}
	defer rows.Close()

	got := []int64{}
	for rows.Next() {
		var labelID int64
		if err := rows.Scan(&labelID); err != nil {
			t.Fatalf("scan label_id: %v", err)
		}
		got = append(got, labelID)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate label_ids: %v", err)
	}
	if len(got) != 2 || got[0] != secondLabelID || got[1] != firstLabelID {
		t.Fatalf("ordered label ids = %v, want [%d %d]", got, secondLabelID, firstLabelID)
	}
}

func TestTargetSchemaRejectsSantaRuleMetadataOnExcludeRows(t *testing.T) {
	db, ctx := dbtest.Open(t)

	labelID := createTargetSchemaLabel(t, ctx, db, "Santa Metadata Label")
	ruleID := createTargetSchemaSantaRule(t, ctx, db)
	_, err := db.Pool().Exec(ctx, `
		INSERT INTO santa_rule_targets (
			rule_id,
			direction,
			position,
			label_id,
			policy,
			cel_expression
		)
		VALUES ($1, 'exclude', 0, $2, 'allowlist', NULL)
	`, ruleID, labelID)
	expectTargetSchemaSQLState(t, err, pgerrcode.CheckViolation)
}

func TestTargetSchemaRestrictsTargetLabelDelete(t *testing.T) {
	db, ctx := dbtest.Open(t)

	labelID := createTargetSchemaLabel(t, ctx, db, "Restrict Delete Label")
	reportID := createTargetSchemaReport(t, ctx, db)
	mustExecTargetSchema(t, ctx, db, `
		INSERT INTO osquery_report_targets (report_id, direction, position, label_id)
		VALUES ($1, 'include', 0, $2)
	`, reportID, labelID)

	_, err := db.Pool().Exec(ctx, `DELETE FROM labels WHERE id = $1`, labelID)
	expectTargetSchemaSQLState(t, err, pgerrcode.RestrictViolation)
}

func TestTargetSchemaRejectsMunkiSoftwareMetadataOnExcludeRows(t *testing.T) {
	db, ctx := dbtest.Open(t)

	labelID := createTargetSchemaLabel(t, ctx, db, "Munki Metadata Label")
	softwareID := createTargetSchemaMunkiSoftware(t, ctx, db)
	_, err := db.Pool().Exec(ctx, `
		INSERT INTO munki_software_targets (
			software_id,
			direction,
			position,
			label_id,
			action,
			optional_install,
			featured_item,
			package_selection
		)
		VALUES ($1, 'exclude', 0, $2, 'install', false, false, 'latest_eligible')
	`, softwareID, labelID)
	expectTargetSchemaSQLState(t, err, pgerrcode.CheckViolation)
}

func createTargetSchemaLabel(
	t testing.TB,
	ctx context.Context,
	db *database.DB,
	name string,
) int64 {
	t.Helper()

	var id int64
	err := db.Pool().QueryRow(ctx, `
		INSERT INTO labels (name, label_type, label_membership_type)
		VALUES ($1, 'regular', 'manual')
		RETURNING id
	`, name).Scan(&id)
	if err != nil {
		t.Fatalf("create label %q: %v", name, err)
	}
	return id
}

func createTargetSchemaCheck(t testing.TB, ctx context.Context, db *database.DB) int64 {
	t.Helper()

	var id int64
	err := db.Pool().QueryRow(ctx, `
		INSERT INTO checks (name, query)
		VALUES ('Schema Check', 'SELECT 1')
		RETURNING id
	`).Scan(&id)
	if err != nil {
		t.Fatalf("create check: %v", err)
	}
	return id
}

func createTargetSchemaReport(t testing.TB, ctx context.Context, db *database.DB) int64 {
	t.Helper()

	var id int64
	err := db.Pool().QueryRow(ctx, `
		INSERT INTO reports (name, query)
		VALUES ('Schema Report', 'SELECT 1')
		RETURNING id
	`).Scan(&id)
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	return id
}

func createTargetSchemaSantaRule(t testing.TB, ctx context.Context, db *database.DB) int64 {
	t.Helper()

	var id int64
	err := db.Pool().QueryRow(ctx, `
		INSERT INTO santa_rules (rule_type, identifier, name)
		VALUES ('teamid', 'ABCDE12345', 'Schema Rule')
		RETURNING id
	`).Scan(&id)
	if err != nil {
		t.Fatalf("create Santa rule: %v", err)
	}
	return id
}

func createTargetSchemaMunkiSoftware(t testing.TB, ctx context.Context, db *database.DB) int64 {
	t.Helper()

	var id int64
	err := db.Pool().QueryRow(ctx, `
		INSERT INTO munki_software_titles (name)
		VALUES ('Schema Munki Software')
		RETURNING id
	`).Scan(&id)
	if err != nil {
		t.Fatalf("create Munki software: %v", err)
	}
	return id
}

func mustExecTargetSchema(
	t testing.TB,
	ctx context.Context,
	db *database.DB,
	query string,
	args ...any,
) {
	t.Helper()

	if _, err := db.Pool().Exec(ctx, query, args...); err != nil {
		t.Fatalf("exec target schema query: %v", err)
	}
}

func expectTargetSchemaSQLState(t testing.TB, err error, code string) {
	t.Helper()

	if got := database.SQLState(err); got != code {
		t.Fatalf("SQLState = %q, want %q from err %v", got, code, err)
	}
}
