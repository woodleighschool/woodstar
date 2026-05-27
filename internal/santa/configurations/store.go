package configurations

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type Store struct {
	db *database.DB
	q  *sqlc.Queries
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db, q: db.Queries()}
}

func (s *Store) ListConfigurations(
	ctx context.Context,
	params ConfigurationListParams,
) ([]Configuration, int, error) {
	where, args := configurationListWhere(params)

	var count int
	if err := s.db.Pool().
		QueryRow(ctx, "SELECT count(*) FROM santa_configurations c "+where, args...).
		Scan(&count); err != nil {
		return nil, 0, err
	}
	query, args, err := configurationListSQL(params, where, args)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[sqlc.SantaConfiguration])
	if err != nil {
		return nil, 0, err
	}

	configurations := make([]Configuration, len(records))
	configurationIDs := make([]int64, len(records))
	for i, record := range records {
		configurations[i] = *configurationFromSQLC(record)
		configurationIDs[i] = record.ID
	}
	if err := s.attachConfigurationLabels(ctx, configurations, configurationIDs); err != nil {
		return nil, 0, err
	}
	return configurations, count, nil
}

func (s *Store) GetConfigurationByID(ctx context.Context, id int64) (*Configuration, error) {
	configuration, err := s.getConfigurationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	configurations := []Configuration{*configuration}
	if err := s.attachConfigurationLabels(ctx, configurations, []int64{configuration.ID}); err != nil {
		return nil, err
	}
	return &configurations[0], nil
}

func (s *Store) getConfigurationByID(ctx context.Context, id int64) (*Configuration, error) {
	row, err := s.q.GetSantaConfigurationByID(ctx, sqlc.GetSantaConfigurationByIDParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return configurationFromSQLC(row), nil
}

func (s *Store) CreateConfiguration(ctx context.Context, params ConfigurationMutation) (*Configuration, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	var configurationID int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		err := validateConfigurationLabelsAvailable(ctx, tx, 0, params.LabelIDs)
		if err != nil {
			return err
		}
		row, err := s.q.WithTx(tx).CreateSantaConfiguration(ctx, createConfigurationParams(params))
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		configurationID = row.ID
		return replaceConfigurationLabels(ctx, tx, configurationID, params.LabelIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.GetConfigurationByID(ctx, configurationID)
}

func (s *Store) UpdateConfiguration(
	ctx context.Context,
	id int64,
	params ConfigurationMutation,
) (*Configuration, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		err := validateConfigurationLabelsAvailable(ctx, tx, id, params.LabelIDs)
		if err != nil {
			return err
		}
		row, err := s.q.WithTx(tx).UpdateSantaConfiguration(ctx, updateConfigurationParams(id, params))
		if errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		} else if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		return replaceConfigurationLabels(ctx, tx, row.ID, params.LabelIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.GetConfigurationByID(ctx, id)
}

func (s *Store) DeleteConfiguration(ctx context.Context, id int64) error {
	_, err := s.q.DeleteSantaConfiguration(ctx, sqlc.DeleteSantaConfigurationParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

// DeleteMany removes multiple Santa configurations. Missing IDs are ignored so repeated bulk actions are idempotent.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	deletedIDs, err := s.q.DeleteSantaConfigurations(ctx, sqlc.DeleteSantaConfigurationsParams{Ids: ids})
	if err != nil {
		return 0, err
	}
	return len(deletedIDs), nil
}

func (s *Store) ReorderConfigurations(ctx context.Context, orderedIDs []int64) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id
			FROM santa_configurations
			ORDER BY position, id
		`)
		if err != nil {
			return err
		}
		currentIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			return err
		}
		if !dbutil.SameInt64Set(orderedIDs, currentIDs) {
			return fmt.Errorf("%w: ordered_ids must exactly match existing configuration IDs", dbutil.ErrInvalidInput)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE santa_configurations c
			SET position = -ordered.position
			FROM unnest($1::bigint[]) WITH ORDINALITY AS ordered(id, position)
			WHERE c.id = ordered.id
		`, orderedIDs); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE santa_configurations
			SET position = -position - 1
		`)
		return err
	})
}

func (s *Store) ResolveConfigurationForHost(ctx context.Context, hostID int64) (*ResolvedConfiguration, error) {
	rows, err := s.db.Pool().Query(ctx, configurationWithMatchedLabelSelectSQL+`
		JOIN santa_configuration_labels cl ON cl.configuration_id = c.id
		JOIN label_membership lm ON lm.label_id = cl.label_id AND lm.host_id = $1
		JOIN labels l ON l.id = cl.label_id
		ORDER BY c.position, c.id, l.name, l.id
		LIMIT 1
	`, hostID)
	if err != nil {
		return nil, err
	}
	record, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[resolvedConfigurationRecord])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil //nolint:nilnil // no matching configuration is represented by a nil result.
	}
	if err != nil {
		return nil, err
	}
	return &ResolvedConfiguration{
		Configuration:   *configurationFromSQLC(record.SantaConfiguration),
		MatchedViaLabel: &LabelMatch{ID: record.LabelID, Name: record.LabelName},
	}, nil
}

func replaceConfigurationLabels(ctx context.Context, tx pgx.Tx, configurationID int64, labelIDs []int64) error {
	if _, err := tx.Exec(
		ctx,
		`DELETE FROM santa_configuration_labels WHERE configuration_id = $1`,
		configurationID,
	); err != nil {
		return err
	}
	if len(labelIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO santa_configuration_labels (label_id, configuration_id)
		SELECT label_id, $1
		FROM unnest($2::bigint[]) AS label_id
	`, configurationID, labelIDs)
	return err
}

func validateConfigurationLabelsAvailable(
	ctx context.Context,
	tx pgx.Tx,
	configurationID int64,
	labelIDs []int64,
) error {
	if len(labelIDs) == 0 {
		return nil
	}

	var conflict ConfigurationLabelConflictError
	err := tx.QueryRow(ctx, `
		SELECT
			cl.label_id,
			c.id,
			c.name
		FROM santa_configuration_labels cl
		JOIN santa_configurations c ON c.id = cl.configuration_id
		WHERE cl.label_id = ANY($1)
			AND c.id <> $2
		ORDER BY cl.label_id
		LIMIT 1
	`, labelIDs, configurationID).Scan(
		&conflict.LabelID,
		&conflict.ConfigurationID,
		&conflict.ConfigurationName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return &conflict
}

func (s *Store) attachConfigurationLabels(
	ctx context.Context,
	configurations []Configuration,
	configurationIDs []int64,
) error {
	if len(configurationIDs) == 0 {
		return nil
	}
	configurationIndexes := make(map[int64]int, len(configurations))
	for i := range configurations {
		configurationIndexes[configurations[i].ID] = i
	}

	rows, err := s.db.Pool().Query(ctx, `
		SELECT configuration_id, label_id
		FROM santa_configuration_labels
		WHERE configuration_id = ANY($1)
		ORDER BY configuration_id, label_id
	`, configurationIDs)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var configurationID int64
		var labelID int64
		if err := rows.Scan(&configurationID, &labelID); err != nil {
			return err
		}
		if i, ok := configurationIndexes[configurationID]; ok {
			configurations[i].LabelIDs = append(configurations[i].LabelIDs, labelID)
		}
	}
	return rows.Err()
}

// Validate enforces cross-field rules that the DB and Huma DTO can't express:
// the removable-media action must pair with non-empty remount flags when set
// to "remount".
func (p ConfigurationMutation) Validate() error {
	if err := validateRemovableMediaPolicy(p.RemovableMediaPolicy, "removable_media_policy"); err != nil {
		return err
	}
	return validateRemovableMediaPolicy(p.EncryptedRemovableMediaPolicy, "encrypted_removable_media_policy")
}

func validateRemovableMediaPolicy(policy RemovableMediaPolicy, name string) error {
	if policy.Action == "" {
		return nil
	}
	if policy.Action == RemovableMediaActionRemount && len(policy.RemountFlags) == 0 {
		return fmt.Errorf(
			"%w: %s.remount_flags are required when action is remount",
			dbutil.ErrInvalidInput,
			name,
		)
	}
	return nil
}

func configurationListWhere(params ConfigurationListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			c.name ILIKE ` + search + `
			OR c.position::text ILIKE ` + search + `
			OR c.client_mode::text ILIKE ` + search + `
			OR c.allowed_path_regex ILIKE ` + search + `
			OR c.blocked_path_regex ILIKE ` + search + `
			OR c.event_detail_url ILIKE ` + search + `
			OR c.event_detail_text ILIKE ` + search + `
		)`)
	}
	return where.Build()
}

func configurationListSQL(
	params ConfigurationListParams,
	where string,
	args []any,
) (string, []any, error) {
	return dbutil.ListQuery{
		SelectSQL:    configurationSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    configurationOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "c.position"}, {SQL: "c.id"}},
		Params:       params.ListParams,
	}.Build()
}

func configurationOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"name":       {SQL: "lower(c.name)"},
		"position":   {SQL: "c.position"},
		"updated_at": {SQL: "c.updated_at"},
	}
}

func createConfigurationParams(configuration ConfigurationMutation) sqlc.CreateSantaConfigurationParams {
	removableMediaAction, removableMediaFlags := removableMediaPolicySQLC(configuration.RemovableMediaPolicy)
	encryptedRemovableMediaAction, encryptedRemovableMediaFlags := removableMediaPolicySQLC(
		configuration.EncryptedRemovableMediaPolicy,
	)
	return sqlc.CreateSantaConfigurationParams{
		Name:                                configuration.Name,
		ClientMode:                          sqlc.SantaClientMode(configuration.ClientMode),
		EnableBundles:                       configuration.EnableBundles,
		EnableTransitiveRules:               configuration.EnableTransitiveRules,
		EnableAllEventUpload:                configuration.EnableAllEventUpload,
		FullSyncIntervalSeconds:             int32(configuration.FullSyncIntervalSeconds),
		BatchSize:                           int32(configuration.BatchSize),
		AllowedPathRegex:                    configuration.AllowedPathRegex,
		BlockedPathRegex:                    configuration.BlockedPathRegex,
		RemovableMediaAction:                removableMediaAction,
		RemovableMediaRemountFlags:          removableMediaFlags,
		EncryptedRemovableMediaAction:       encryptedRemovableMediaAction,
		EncryptedRemovableMediaRemountFlags: encryptedRemovableMediaFlags,
		EventDetailURL:                      configuration.EventDetailURL,
		EventDetailText:                     configuration.EventDetailText,
	}
}

func updateConfigurationParams(id int64, configuration ConfigurationMutation) sqlc.UpdateSantaConfigurationParams {
	params := createConfigurationParams(configuration)
	return sqlc.UpdateSantaConfigurationParams{
		Name:                                params.Name,
		ClientMode:                          params.ClientMode,
		EnableBundles:                       params.EnableBundles,
		EnableTransitiveRules:               params.EnableTransitiveRules,
		EnableAllEventUpload:                params.EnableAllEventUpload,
		FullSyncIntervalSeconds:             params.FullSyncIntervalSeconds,
		BatchSize:                           params.BatchSize,
		AllowedPathRegex:                    params.AllowedPathRegex,
		BlockedPathRegex:                    params.BlockedPathRegex,
		RemovableMediaAction:                params.RemovableMediaAction,
		RemovableMediaRemountFlags:          params.RemovableMediaRemountFlags,
		EncryptedRemovableMediaAction:       params.EncryptedRemovableMediaAction,
		EncryptedRemovableMediaRemountFlags: params.EncryptedRemovableMediaRemountFlags,
		EventDetailURL:                      params.EventDetailURL,
		EventDetailText:                     params.EventDetailText,
		ID:                                  id,
	}
}

func configurationFromSQLC(row sqlc.SantaConfiguration) *Configuration {
	return &Configuration{
		ID:                      row.ID,
		Name:                    row.Name,
		Position:                int(row.Position),
		ClientMode:              ClientMode(row.ClientMode),
		EnableBundles:           row.EnableBundles,
		EnableTransitiveRules:   row.EnableTransitiveRules,
		EnableAllEventUpload:    row.EnableAllEventUpload,
		FullSyncIntervalSeconds: int(row.FullSyncIntervalSeconds),
		BatchSize:               int(row.BatchSize),
		AllowedPathRegex:        row.AllowedPathRegex,
		BlockedPathRegex:        row.BlockedPathRegex,
		RemovableMediaPolicy: removableMediaPolicyFromSQLC(
			row.RemovableMediaAction,
			row.RemovableMediaRemountFlags,
		),
		EncryptedRemovableMediaPolicy: removableMediaPolicyFromSQLC(
			row.EncryptedRemovableMediaAction,
			row.EncryptedRemovableMediaRemountFlags,
		),
		EventDetailURL:  row.EventDetailURL,
		EventDetailText: row.EventDetailText,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func removableMediaPolicySQLC(policy RemovableMediaPolicy) (*sqlc.SantaRemovableMediaAction, []string) {
	if policy.Action == "" {
		return nil, nil
	}
	action := sqlc.SantaRemovableMediaAction(policy.Action)
	return &action, policy.RemountFlags
}

func removableMediaPolicyFromSQLC(action *sqlc.SantaRemovableMediaAction, flags []string) RemovableMediaPolicy {
	if action == nil {
		return RemovableMediaPolicy{}
	}
	return RemovableMediaPolicy{
		Action:       RemovableMediaAction(*action),
		RemountFlags: flags,
	}
}

type resolvedConfigurationRecord struct {
	sqlc.SantaConfiguration
	LabelID   int64  `db:"label_id"`
	LabelName string `db:"label_name"`
}

const configurationSelectSQL = `
SELECT
	c.id,
	c.name,
	c.position,
	c.client_mode,
	c.enable_bundles,
	c.enable_transitive_rules,
	c.enable_all_event_upload,
	c.full_sync_interval_seconds,
	c.batch_size,
	c.allowed_path_regex,
	c.blocked_path_regex,
	c.removable_media_action,
	c.removable_media_remount_flags,
	c.encrypted_removable_media_action,
	c.encrypted_removable_media_remount_flags,
	c.event_detail_url,
	c.event_detail_text,
	c.created_at,
	c.updated_at
FROM santa_configurations c`

const configurationWithMatchedLabelSelectSQL = `
SELECT
	c.id,
	c.name,
	c.position,
	c.client_mode,
	c.enable_bundles,
	c.enable_transitive_rules,
	c.enable_all_event_upload,
	c.full_sync_interval_seconds,
	c.batch_size,
	c.allowed_path_regex,
	c.blocked_path_regex,
	c.removable_media_action,
	c.removable_media_remount_flags,
	c.encrypted_removable_media_action,
	c.encrypted_removable_media_remount_flags,
	c.event_detail_url,
	c.event_detail_text,
	c.created_at,
	c.updated_at,
	l.id AS label_id,
	l.name AS label_name
FROM santa_configurations c`
