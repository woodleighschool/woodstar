package configurations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/targeting"
)

type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(
	ctx context.Context,
	params ConfigurationListParams,
) ([]Configuration, int, error) {
	where, args := configurationListWhere(params)
	listQuery := dbutil.ListQuery{
		SelectSQL:    configurationSelectSQL(),
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    configurationOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "c.position"}, {SQL: "c.id"}},
		Params:       params.ListParams,
	}

	rows, count, err := dbutil.ListWithCount[configurationRow](ctx, s.db.Pool(), listQuery)
	if err != nil {
		return nil, 0, err
	}

	configurations := make([]Configuration, len(rows))
	configurationIDs := make([]int64, len(rows))
	for i, row := range rows {
		configurations[i] = configurationFromRow(row)
		configurationIDs[i] = row.ID
	}
	if err := s.attachConfigurationTargets(ctx, configurations, configurationIDs); err != nil {
		return nil, 0, err
	}
	return configurations, count, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (*Configuration, error) {
	if id <= 0 {
		return nil, dbutil.ErrNotFound
	}
	row, err := dbutil.GetOne[configurationRow](
		ctx,
		s.db.Pool(),
		configurationSelectSQL()+"\nWHERE c.id = $1",
		id,
	)
	if err != nil {
		return nil, err
	}
	configuration := configurationFromRow(row)
	configurations := []Configuration{configuration}
	if err := s.attachConfigurationTargets(ctx, configurations, []int64{configuration.ID}); err != nil {
		return nil, err
	}
	return &configurations[0], nil
}

func (s *Store) Create(ctx context.Context, params ConfigurationMutation) (*Configuration, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	write := newConfigurationWrite(params)
	var id int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO santa_configurations (
				name,
				description,
				position,
				client_mode,
				enable_bundles,
				enable_transitive_rules,
				enable_all_event_upload,
				disable_unknown_event_upload,
				override_file_access_action,
				full_sync_interval_seconds,
				batch_size,
				allowed_path_regex,
				blocked_path_regex,
				removable_media_action,
				removable_media_remount_flags,
				encrypted_removable_media_action,
				encrypted_removable_media_remount_flags,
				event_detail_url,
				event_detail_text
			) VALUES (
				@name,
				@description,
				(SELECT COALESCE(MAX(position) + 1, 0) FROM santa_configurations),
				@client_mode::santa_client_mode,
				@enable_bundles,
				@enable_transitive_rules,
				@enable_all_event_upload,
				@disable_unknown_event_upload,
				@override_file_access_action::santa_file_access_action,
				@full_sync_interval_seconds::integer,
				@batch_size::integer,
				@allowed_path_regex,
				@blocked_path_regex,
				@removable_media_action::santa_removable_media_action,
				@removable_media_remount_flags::text[],
				@encrypted_removable_media_action::santa_removable_media_action,
				@encrypted_removable_media_remount_flags::text[],
				@event_detail_url,
				@event_detail_text
			)
			RETURNING id`, pgx.StructArgs(write)).Scan(&id); err != nil {
			return dbutil.MutationError(err)
		}
		return replaceConfigurationTargets(ctx, tx, id, params.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Update(ctx context.Context, id int64, params ConfigurationMutation) (*Configuration, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	write := newConfigurationWrite(params)
	write.ID = id
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var updatedID int64
		if err := tx.QueryRow(ctx, `
			UPDATE santa_configurations
			SET
				name = @name,
				description = @description,
				client_mode = @client_mode::santa_client_mode,
				enable_bundles = @enable_bundles,
				enable_transitive_rules = @enable_transitive_rules,
				enable_all_event_upload = @enable_all_event_upload,
				disable_unknown_event_upload = @disable_unknown_event_upload,
				override_file_access_action = @override_file_access_action::santa_file_access_action,
				full_sync_interval_seconds = @full_sync_interval_seconds::integer,
				batch_size = @batch_size::integer,
				allowed_path_regex = @allowed_path_regex,
				blocked_path_regex = @blocked_path_regex,
				removable_media_action = @removable_media_action::santa_removable_media_action,
				removable_media_remount_flags = @removable_media_remount_flags::text[],
				encrypted_removable_media_action = @encrypted_removable_media_action::santa_removable_media_action,
				encrypted_removable_media_remount_flags = @encrypted_removable_media_remount_flags::text[],
				event_detail_url = @event_detail_url,
				event_detail_text = @event_detail_text,
				updated_at = now()
			WHERE id = @id
			RETURNING id`, pgx.StructArgs(write)).Scan(&updatedID); err != nil {
			return dbutil.MutationError(err)
		}
		return replaceConfigurationTargets(ctx, tx, id, params.Targets)
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	tag, err := s.db.Pool().Exec(ctx, `DELETE FROM santa_configurations WHERE id = $1`, id)
	if err != nil {
		return dbutil.DeleteConflict(err, "Santa configuration is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return dbutil.ErrNotFound
	}
	return nil
}

// DeleteMany removes multiple Santa configurations. Missing IDs are ignored so repeated bulk actions are idempotent.
func (s *Store) DeleteMany(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.db.Pool().Query(
		ctx,
		`DELETE FROM santa_configurations WHERE id = ANY($1::bigint[]) RETURNING id`,
		ids,
	)
	if err != nil {
		return 0, err
	}
	deletedIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, err
	}
	return len(deletedIDs), nil
}

func (s *Store) ReorderConfigurations(ctx context.Context, orderedIDs []int64) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		var updated, total int
		if err := tx.QueryRow(ctx, `
WITH ordered AS (
	SELECT id, position::int
	FROM unnest($1::bigint[]) WITH ORDINALITY AS input(id, position)
),
stats AS (
	SELECT
		(SELECT count(*) FROM santa_configurations) AS total,
		(SELECT count(*) FROM ordered) AS requested,
		(SELECT count(DISTINCT id) FROM ordered) AS distinct_requested,
		(
			SELECT count(*)
			FROM ordered
			JOIN santa_configurations c ON c.id = ordered.id
		) AS matched
),
updated AS (
	UPDATE santa_configurations c
	SET position = -ordered.position
	FROM ordered, stats
	WHERE stats.total = stats.requested
	  AND stats.requested = stats.distinct_requested
	  AND stats.requested = stats.matched
	  AND c.id = ordered.id
	RETURNING c.id
)
SELECT (SELECT count(*) FROM updated), (SELECT total FROM stats)`,
			orderedIDs,
		).Scan(&updated, &total); err != nil {
			return err
		}
		if updated != total {
			return fmt.Errorf("%w: ordered_ids must exactly match existing configuration IDs", dbutil.ErrInvalidInput)
		}
		_, err := tx.Exec(ctx, `UPDATE santa_configurations SET position = -position - 1`)
		return err
	})
}

func (s *Store) ResolveConfigurationForHost(ctx context.Context, hostID int64) (*ConfigurationMatch, error) {
	type resolveRow struct {
		configurationRow

		LabelID   int64  `db:"label_id"`
		LabelName string `db:"label_name"`
	}

	qrows, err := s.db.Pool().Query(ctx, `
		SELECT
			c.id,
			c.name,
			c.description,
			c.position,
			c.client_mode,
			c.enable_bundles,
			c.enable_transitive_rules,
			c.enable_all_event_upload,
			c.disable_unknown_event_upload,
			c.override_file_access_action::text AS override_file_access_action,
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
		FROM santa_configurations c
		JOIN LATERAL (
			SELECT
				include_label.id,
				include_label.name
			FROM santa_configuration_targets t
			JOIN label_membership lm ON lm.label_id = t.label_id AND lm.host_id = $1
			JOIN labels include_label ON include_label.id = t.label_id
			WHERE t.configuration_id = c.id
				AND t.direction = 'include'
			ORDER BY t.position
			LIMIT 1
		) l ON true
		WHERE NOT EXISTS (
			SELECT 1
			FROM santa_configuration_targets t
			JOIN label_membership lm ON lm.label_id = t.label_id AND lm.host_id = $1
			WHERE t.configuration_id = c.id
				AND t.direction = 'exclude'
		)
		ORDER BY c.position, c.id
		LIMIT 1`, hostID)
	if err != nil {
		return nil, err
	}
	rr, err := pgx.CollectExactlyOneRow(qrows, pgx.RowToStructByName[resolveRow])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	cfg := configurationFromRow(rr.configurationRow)
	return &ConfigurationMatch{
		Configuration:   cfg,
		MatchedViaLabel: &LabelMatch{ID: rr.LabelID, Name: rr.LabelName},
	}, nil
}

func (s *Store) ResolveConfigurationForHostWithTargets(ctx context.Context, hostID int64) (*ConfigurationMatch, error) {
	match, err := s.ResolveConfigurationForHost(ctx, hostID)
	if err != nil || match == nil {
		return match, err
	}
	configuration := match.Configuration
	configurations := []Configuration{configuration}
	if err := s.attachConfigurationTargets(ctx, configurations, []int64{configuration.ID}); err != nil {
		return nil, err
	}
	match.Configuration = configurations[0]
	return match, nil
}

type configurationTargetWrite struct {
	ConfigurationID int64  `db:"configuration_id"`
	LabelID         int64  `db:"label_id"`
	Direction       string `db:"direction"`
	Position        int32  `db:"position"`
}

func replaceConfigurationTargets(
	ctx context.Context,
	tx pgx.Tx,
	configurationID int64,
	targets ConfigurationTargets,
) error {
	targets = normalizeConfigurationTargets(targets)
	rows := make([]configurationTargetWrite, 0, len(targets.Include)+len(targets.Exclude))
	for i, ref := range targets.Include {
		rows = append(rows, configurationTargetWrite{
			ConfigurationID: configurationID,
			LabelID:         ref.LabelID,
			Direction:       string(targeting.Include),
			Position:        int32(i),
		})
	}
	for i, ref := range targets.Exclude {
		rows = append(rows, configurationTargetWrite{
			ConfigurationID: configurationID,
			LabelID:         ref.LabelID,
			Direction:       string(targeting.Exclude),
			Position:        int32(i),
		})
	}
	if err := dbutil.ReplaceChildren(
		ctx, tx,
		`DELETE FROM santa_configuration_targets WHERE configuration_id = $1`, []any{configurationID},
		`
			INSERT INTO santa_configuration_targets (configuration_id, label_id, direction, position)
			VALUES (@configuration_id, @label_id, @direction::target_direction, @position)`, rows,
	); err != nil {
		return dbutil.MutationError(err)
	}
	return nil
}

func (s *Store) attachConfigurationTargets(
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
		configurations[i].Targets = emptyConfigurationTargets()
	}

	type targetRow struct {
		ConfigurationID int64  `db:"configuration_id"`
		LabelID         int64  `db:"label_id"`
		Direction       string `db:"direction"`
	}

	qrows, err := s.db.Pool().Query(ctx, `
		SELECT configuration_id, label_id, direction::text AS direction
		FROM santa_configuration_targets
		WHERE configuration_id = ANY($1::bigint[])
		ORDER BY configuration_id, direction, position`,
		configurationIDs,
	)
	if err != nil {
		return err
	}
	rows, err := pgx.CollectRows(qrows, pgx.RowToStructByName[targetRow])
	if err != nil {
		return err
	}

	for _, row := range rows {
		if i, ok := configurationIndexes[row.ConfigurationID]; ok {
			targetSet := configurations[i].Targets
			ref := targeting.LabelRef{LabelID: row.LabelID}
			switch targeting.Direction(row.Direction) {
			case targeting.Include:
				targetSet.Include = append(targetSet.Include, ref)
			case targeting.Exclude:
				targetSet.Exclude = append(targetSet.Exclude, ref)
			default:
				return fmt.Errorf("%w: unsupported target direction %q", dbutil.ErrInvalidInput, row.Direction)
			}
			configurations[i].Targets = targetSet
		}
	}
	return nil
}

func configurationListWhere(params ConfigurationListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add(`(
			c.name ILIKE ` + search + `
			OR c.description ILIKE ` + search + `
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

func configurationOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"name":        {SQL: "lower(c.name)"},
		"description": {SQL: "lower(c.description)"},
		"position":    {SQL: "c.position"},
		"updated_at":  {SQL: "c.updated_at"},
	}
}

type configurationRow struct {
	ID                                  int64     `db:"id"`
	Name                                string    `db:"name"`
	Description                         string    `db:"description"`
	Position                            int32     `db:"position"`
	ClientMode                          string    `db:"client_mode"`
	EnableBundles                       bool      `db:"enable_bundles"`
	EnableTransitiveRules               bool      `db:"enable_transitive_rules"`
	EnableAllEventUpload                bool      `db:"enable_all_event_upload"`
	DisableUnknownEventUpload           bool      `db:"disable_unknown_event_upload"`
	OverrideFileAccessAction            string    `db:"override_file_access_action"`
	FullSyncIntervalSeconds             int32     `db:"full_sync_interval_seconds"`
	BatchSize                           int32     `db:"batch_size"`
	AllowedPathRegex                    string    `db:"allowed_path_regex"`
	BlockedPathRegex                    string    `db:"blocked_path_regex"`
	RemovableMediaAction                *string   `db:"removable_media_action"`
	RemovableMediaRemountFlags          []string  `db:"removable_media_remount_flags"`
	EncryptedRemovableMediaAction       *string   `db:"encrypted_removable_media_action"`
	EncryptedRemovableMediaRemountFlags []string  `db:"encrypted_removable_media_remount_flags"`
	EventDetailURL                      string    `db:"event_detail_url"`
	EventDetailText                     string    `db:"event_detail_text"`
	CreatedAt                           time.Time `db:"created_at"`
	UpdatedAt                           time.Time `db:"updated_at"`
}

func configurationFromRow(row configurationRow) Configuration {
	return Configuration{
		ID:                        row.ID,
		Name:                      row.Name,
		Description:               row.Description,
		Position:                  row.Position,
		ClientMode:                ClientMode(row.ClientMode),
		EnableBundles:             row.EnableBundles,
		EnableTransitiveRules:     row.EnableTransitiveRules,
		EnableAllEventUpload:      row.EnableAllEventUpload,
		DisableUnknownEventUpload: row.DisableUnknownEventUpload,
		OverrideFileAccessAction:  FileAccessAction(row.OverrideFileAccessAction),
		FullSyncIntervalSeconds:   row.FullSyncIntervalSeconds,
		BatchSize:                 row.BatchSize,
		AllowedPathRegex:          row.AllowedPathRegex,
		BlockedPathRegex:          row.BlockedPathRegex,
		RemovableMediaPolicy: removableMediaPolicyFromRow(
			row.RemovableMediaAction,
			row.RemovableMediaRemountFlags,
		),
		EncryptedRemovableMediaPolicy: removableMediaPolicyFromRow(
			row.EncryptedRemovableMediaAction,
			row.EncryptedRemovableMediaRemountFlags,
		),
		EventDetailURL:  row.EventDetailURL,
		EventDetailText: row.EventDetailText,
		Targets:         emptyConfigurationTargets(),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func removableMediaPolicyFromRow(action *string, flags []string) RemovableMediaPolicy {
	if action == nil {
		return RemovableMediaPolicy{}
	}
	return RemovableMediaPolicy{
		Action:       RemovableMediaAction(*action),
		RemountFlags: flags,
	}
}

type configurationWrite struct {
	ID                                  int64    `db:"id"`
	Name                                string   `db:"name"`
	Description                         string   `db:"description"`
	ClientMode                          string   `db:"client_mode"`
	EnableBundles                       bool     `db:"enable_bundles"`
	EnableTransitiveRules               bool     `db:"enable_transitive_rules"`
	EnableAllEventUpload                bool     `db:"enable_all_event_upload"`
	DisableUnknownEventUpload           bool     `db:"disable_unknown_event_upload"`
	OverrideFileAccessAction            string   `db:"override_file_access_action"`
	FullSyncIntervalSeconds             int32    `db:"full_sync_interval_seconds"`
	BatchSize                           int32    `db:"batch_size"`
	AllowedPathRegex                    string   `db:"allowed_path_regex"`
	BlockedPathRegex                    string   `db:"blocked_path_regex"`
	RemovableMediaAction                *string  `db:"removable_media_action"`
	RemovableMediaRemountFlags          []string `db:"removable_media_remount_flags"`
	EncryptedRemovableMediaAction       *string  `db:"encrypted_removable_media_action"`
	EncryptedRemovableMediaRemountFlags []string `db:"encrypted_removable_media_remount_flags"`
	EventDetailURL                      string   `db:"event_detail_url"`
	EventDetailText                     string   `db:"event_detail_text"`
}

func newConfigurationWrite(p ConfigurationMutation) configurationWrite {
	rma, rmf := removableMediaWriteFields(p.RemovableMediaPolicy)
	erma, ermf := removableMediaWriteFields(p.EncryptedRemovableMediaPolicy)
	return configurationWrite{
		Name:                                p.Name,
		Description:                         p.Description,
		ClientMode:                          string(p.ClientMode),
		EnableBundles:                       p.EnableBundles,
		EnableTransitiveRules:               p.EnableTransitiveRules,
		EnableAllEventUpload:                p.EnableAllEventUpload,
		DisableUnknownEventUpload:           p.DisableUnknownEventUpload,
		OverrideFileAccessAction:            string(p.OverrideFileAccessAction),
		FullSyncIntervalSeconds:             p.FullSyncIntervalSeconds,
		BatchSize:                           p.BatchSize,
		AllowedPathRegex:                    p.AllowedPathRegex,
		BlockedPathRegex:                    p.BlockedPathRegex,
		RemovableMediaAction:                rma,
		RemovableMediaRemountFlags:          rmf,
		EncryptedRemovableMediaAction:       erma,
		EncryptedRemovableMediaRemountFlags: ermf,
		EventDetailURL:                      p.EventDetailURL,
		EventDetailText:                     p.EventDetailText,
	}
}

func removableMediaWriteFields(policy RemovableMediaPolicy) (*string, []string) {
	if policy.Action == "" {
		return nil, nil
	}
	action := string(policy.Action)
	return &action, policy.RemountFlags
}

func configurationSelectSQL() string {
	return `
SELECT
	c.id,
	c.name,
	c.description,
	c.position,
	c.client_mode,
	c.enable_bundles,
	c.enable_transitive_rules,
	c.enable_all_event_upload,
	c.disable_unknown_event_upload,
	c.override_file_access_action::text AS override_file_access_action,
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
}
