package syncstate

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/postgres"
)

const (
	phaseDesired = "desired"
	phaseApplied = "applied"
)

func loadTargets(ctx context.Context, q postgres.Queryer, hostID int64, phase string) ([]Target, error) {
	rows, err := q.Query(ctx, `
SELECT rule_type::text, identifier, policy::text, cel_expression,
       custom_message, custom_url, notification_app_name
FROM santa_sync_targets
WHERE host_id = $1 AND phase = $2::santa_sync_target_phase`, hostID, phase)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Target])
}

func applyTargetChanges(ctx context.Context, tx pgx.Tx, hostID int64, phase string, changes []PayloadRule) error {
	if len(changes) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
WITH changes AS (
    SELECT rule_type, identifier, policy, cel_expression, custom_message, custom_url, notification_app_name, removed
    FROM jsonb_to_recordset($3) AS c(
        rule_type santa_rule_type, identifier text, policy santa_policy, cel_expression text,
        custom_message text, custom_url text, notification_app_name text, removed boolean
    )
), removed AS (
    DELETE FROM santa_sync_targets t
    USING changes c
    WHERE t.host_id = $1 AND t.phase = $2::santa_sync_target_phase
      AND (t.rule_type, t.identifier) = (c.rule_type, c.identifier) AND c.removed
)
INSERT INTO santa_sync_targets (
    host_id, phase, rule_type, identifier, policy,
    cel_expression, custom_message, custom_url, notification_app_name
)
SELECT $1, $2::santa_sync_target_phase, rule_type, identifier, policy,
       COALESCE(cel_expression, ''), COALESCE(custom_message, ''), COALESCE(custom_url, ''),
       COALESCE(notification_app_name, '')
FROM changes WHERE removed IS NOT TRUE
ON CONFLICT (host_id, phase, rule_type, identifier) DO UPDATE SET
    policy = EXCLUDED.policy, cel_expression = EXCLUDED.cel_expression,
    custom_message = EXCLUDED.custom_message, custom_url = EXCLUDED.custom_url,
    notification_app_name = EXCLUDED.notification_app_name, updated_at = now()`, hostID, phase, changes)
	return err
}
