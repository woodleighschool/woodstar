package syncstate

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

const (
	phaseDesired = "desired"
	phaseApplied = "applied"
)

type targetRow struct {
	RuleType            string `db:"rule_type"`
	Identifier          string `db:"identifier"`
	Policy              string `db:"policy"`
	CelExpression       string `db:"cel_expression"`
	CustomMessage       string `db:"custom_message"`
	CustomURL           string `db:"custom_url"`
	NotificationAppName string `db:"notification_app_name"`
	PayloadHash         string `db:"payload_hash"`
}

const listTargetsSQL = `
SELECT
    rule_type::text,
    identifier,
    policy::text,
    cel_expression,
    custom_message,
    custom_url,
    notification_app_name,
    payload_hash
FROM santa_sync_targets
WHERE host_id = $1 AND phase = $2::santa_sync_target_phase
ORDER BY position`

func loadTargets(ctx context.Context, q dbutil.Queryer, hostID int64, phase string) ([]Target, error) {
	qrows, err := q.Query(ctx, listTargetsSQL, hostID, phase)
	if err != nil {
		return nil, err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[targetRow])
	if err != nil {
		return nil, err
	}
	targets := make([]Target, len(rows))
	for i, row := range rows {
		targets[i] = targetFromRow(row)
	}
	return sortedTargets(targets), nil
}

func insertTargets(ctx context.Context, tx pgx.Tx, hostID int64, phase string, targets []Target) error {
	for position, target := range sortedTargets(targets) {
		if _, err := tx.Exec(ctx, `
INSERT INTO santa_sync_targets (
    host_id, phase, position, rule_type, identifier, policy,
    cel_expression, custom_message, custom_url, notification_app_name, payload_hash
)
VALUES (
    $1, $2::santa_sync_target_phase, $3, $4::santa_rule_type, $5, $6::santa_policy,
    $7, $8, $9, $10, $11
)`,
			hostID, phase, int32(position),
			target.RuleType, target.Identifier, target.Policy,
			target.CELExpression, target.CustomMessage, target.CustomURL,
			target.AppName, target.PayloadHash,
		); err != nil {
			return err
		}
	}
	return nil
}

func targetFromRow(row targetRow) Target {
	return Target{
		RuleType:      row.RuleType,
		Identifier:    row.Identifier,
		Policy:        row.Policy,
		CELExpression: row.CelExpression,
		CustomMessage: row.CustomMessage,
		CustomURL:     row.CustomURL,
		AppName:       row.NotificationAppName,
		PayloadHash:   row.PayloadHash,
	}
}
