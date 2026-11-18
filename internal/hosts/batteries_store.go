package hosts

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/postgres"
)

func (s *Store) ReplaceBatteries(ctx context.Context, hostID int64, batteries []HostBattery) error {
	return s.reconcileHostInventory(ctx, hostID, reconcileBatteriesSQL, postgres.JSONSlice[HostBattery](batteries))
}

const reconcileBatteriesSQL = `
WITH observed AS (
    SELECT DISTINCT ON (serial_number) *
    FROM jsonb_populate_recordset(NULL::host_batteries, $2::jsonb) WITH ORDINALITY
    WHERE serial_number <> ''
    ORDER BY serial_number, ordinality DESC
), removed AS (
    DELETE FROM host_batteries t
    WHERE t.host_id = $1
      AND NOT EXISTS (SELECT FROM observed s WHERE t.serial_number = s.serial_number)
)
MERGE INTO host_batteries t
USING observed s ON t.host_id = $1 AND t.serial_number = s.serial_number
WHEN MATCHED AND (
    t.manufacturer, t.model, t.chemistry, t.cycle_count, t.health, t.designed_capacity,
    t.max_capacity, t.current_capacity, t.percent_remaining
) IS DISTINCT FROM (
    s.manufacturer, s.model, s.chemistry, s.cycle_count, s.health, s.designed_capacity,
    s.max_capacity, s.current_capacity, s.percent_remaining
) THEN UPDATE SET
    (manufacturer, model, chemistry, cycle_count, health, designed_capacity, max_capacity,
     current_capacity, percent_remaining) =
    (s.manufacturer, s.model, s.chemistry, s.cycle_count, s.health, s.designed_capacity,
     s.max_capacity, s.current_capacity, s.percent_remaining)
WHEN NOT MATCHED THEN INSERT (
    host_id, serial_number, manufacturer, model, chemistry, cycle_count, health,
    designed_capacity, max_capacity, current_capacity, percent_remaining
) VALUES (
    $1, s.serial_number, s.manufacturer, s.model, s.chemistry, s.cycle_count, s.health,
    s.designed_capacity, s.max_capacity, s.current_capacity, s.percent_remaining
)`

func (s *Store) ListBatteries(ctx context.Context, hostID int64) ([]HostBattery, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
	id, host_id, serial_number, manufacturer, model, chemistry, cycle_count,
	health, designed_capacity, max_capacity, current_capacity, percent_remaining
FROM host_batteries
WHERE host_id = $1
ORDER BY serial_number, id`,
		hostID,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[HostBattery])
}
