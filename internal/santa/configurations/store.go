package configurations

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/database/sqlc"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/targeting"
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
	listQuery := configurationListQuery(params, where, args)

	records, count, err := dbutil.ListWithCount[sqlc.SantaConfiguration](
		ctx,
		s.db.Pool(),
		listQuery,
	)
	if err != nil {
		return nil, 0, err
	}

	configurations := make([]Configuration, len(records))
	configurationIDs := make([]int64, len(records))
	for i, record := range records {
		configurations[i] = *configurationFromSQLC(record)
		configurationIDs[i] = record.ID
	}
	if err := s.attachConfigurationTargets(ctx, configurations, configurationIDs); err != nil {
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
	if err := s.attachConfigurationTargets(ctx, configurations, []int64{configuration.ID}); err != nil {
		return nil, err
	}
	return &configurations[0], nil
}

func (s *Store) getConfigurationByID(ctx context.Context, id int64) (*Configuration, error) {
	row, err := s.q.GetSantaConfigurationByID(ctx, sqlc.GetSantaConfigurationByIDParams{ID: id})
	if err != nil {
		return nil, dbutil.GetError(err)
	}
	return configurationFromSQLC(row), nil
}

func (s *Store) CreateConfiguration(ctx context.Context, params ConfigurationMutation) (*Configuration, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	var configurationID int64
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		row, err := s.q.WithTx(tx).CreateSantaConfiguration(ctx, createConfigurationParams(params))
		if err != nil {
			return dbutil.MutationError(err)
		}
		configurationID = row.ID
		if err := replaceConfigurationTargets(ctx, tx, configurationID, params.Targets); err != nil {
			return dbutil.MutationError(err)
		}
		return nil
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
		row, err := s.q.WithTx(tx).UpdateSantaConfiguration(ctx, updateConfigurationParams(id, params))
		if err != nil {
			return dbutil.MutationError(err)
		}
		if err := replaceConfigurationTargets(ctx, tx, row.ID, params.Targets); err != nil {
			return dbutil.MutationError(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetConfigurationByID(ctx, id)
}

func (s *Store) DeleteConfiguration(ctx context.Context, id int64) error {
	_, err := s.q.DeleteSantaConfiguration(ctx, sqlc.DeleteSantaConfigurationParams{ID: id})
	return dbutil.GetError(err)
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
		q := s.q.WithTx(tx)
		currentIDs, err := q.ListSantaConfigurationIDsByPosition(ctx)
		if err != nil {
			return err
		}
		if !dbutil.SameInt64Set(orderedIDs, currentIDs) {
			return fmt.Errorf("%w: ordered_ids must exactly match existing configuration IDs", dbutil.ErrInvalidInput)
		}
		if err := q.SetSantaConfigurationPositions(
			ctx,
			sqlc.SetSantaConfigurationPositionsParams{OrderedIds: orderedIDs},
		); err != nil {
			return err
		}
		return q.NormalizeSantaConfigurationPositions(ctx)
	})
}

func (s *Store) ResolveConfigurationForHost(ctx context.Context, hostID int64) (*ConfigurationMatch, error) {
	record, err := s.q.ResolveSantaConfigurationForHost(
		ctx,
		sqlc.ResolveSantaConfigurationForHostParams{HostID: hostID},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	configuration := *configurationFromSQLC(record.SantaConfiguration)
	return &ConfigurationMatch{
		Configuration:   configuration,
		MatchedViaLabel: &LabelMatch{ID: record.LabelID, Name: record.LabelName},
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

func replaceConfigurationTargets(
	ctx context.Context,
	tx pgx.Tx,
	configurationID int64,
	targets ConfigurationTargets,
) error {
	targets = normalizeConfigurationTargets(targets)
	if err := targets.validate(); err != nil {
		return err
	}
	q := sqlc.New(tx)
	if err := q.DeleteSantaConfigurationTargets(
		ctx,
		sqlc.DeleteSantaConfigurationTargetsParams{ConfigurationID: configurationID},
	); err != nil {
		return err
	}
	if len(targets.Include) > 0 {
		if err := q.InsertSantaConfigurationTargets(ctx, sqlc.InsertSantaConfigurationTargetsParams{
			ConfigurationID: configurationID,
			LabelIds:        labelRefIDs(targets.Include),
			Direction:       sqlc.TargetDirection(targeting.Include),
		}); err != nil {
			return err
		}
	}
	if len(targets.Exclude) > 0 {
		if err := q.InsertSantaConfigurationTargets(ctx, sqlc.InsertSantaConfigurationTargetsParams{
			ConfigurationID: configurationID,
			LabelIds:        labelRefIDs(targets.Exclude),
			Direction:       sqlc.TargetDirection(targeting.Exclude),
		}); err != nil {
			return err
		}
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

	rows, err := s.q.ListSantaConfigurationTargets(
		ctx,
		sqlc.ListSantaConfigurationTargetsParams{ConfigurationIds: configurationIDs},
	)
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

// Validate enforces caller-facing rules before storage.
func (p ConfigurationMutation) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if !slices.Contains(ClientModeValues, p.ClientMode) {
		return fmt.Errorf("%w: client_mode is required", dbutil.ErrInvalidInput)
	}
	if p.FullSyncIntervalSeconds < 60 {
		return fmt.Errorf("%w: full_sync_interval_seconds must be at least 60", dbutil.ErrInvalidInput)
	}
	if p.BatchSize < 5 || p.BatchSize > 100 {
		return fmt.Errorf("%w: batch_size must be between 5 and 100", dbutil.ErrInvalidInput)
	}
	if err := validateRemovableMediaPolicy(p.RemovableMediaPolicy, "removable_media_policy"); err != nil {
		return err
	}
	if err := p.Targets.validate(); err != nil {
		return err
	}
	return validateRemovableMediaPolicy(p.EncryptedRemovableMediaPolicy, "encrypted_removable_media_policy")
}

func validateRemovableMediaPolicy(policy RemovableMediaPolicy, name string) error {
	if policy.Action == "" {
		return nil
	}
	if !slices.Contains(RemovableMediaActionValues, policy.Action) {
		return fmt.Errorf("%w: %s.action is invalid", dbutil.ErrInvalidInput, name)
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

func configurationListQuery(
	params ConfigurationListParams,
	where string,
	args []any,
) dbutil.ListQuery {
	return dbutil.ListQuery{
		SelectSQL:    configurationSelectSQL,
		WhereSQL:     where,
		Args:         args,
		OrderKeys:    configurationOrderKeys(),
		DefaultOrder: []dbutil.OrderExpr{{SQL: "c.position"}, {SQL: "c.id"}},
		Params:       params.ListParams,
	}
}

func configurationOrderKeys() map[string]dbutil.OrderExpr {
	return map[string]dbutil.OrderExpr{
		"name":        {SQL: "lower(c.name)"},
		"description": {SQL: "lower(c.description)"},
		"position":    {SQL: "c.position"},
		"updated_at":  {SQL: "c.updated_at"},
	}
}

func createConfigurationParams(configuration ConfigurationMutation) sqlc.CreateSantaConfigurationParams {
	removableMediaAction, removableMediaFlags := removableMediaPolicySQLC(configuration.RemovableMediaPolicy)
	encryptedRemovableMediaAction, encryptedRemovableMediaFlags := removableMediaPolicySQLC(
		configuration.EncryptedRemovableMediaPolicy,
	)
	return sqlc.CreateSantaConfigurationParams{
		Name:                                configuration.Name,
		Description:                         configuration.Description,
		ClientMode:                          sqlc.SantaClientMode(configuration.ClientMode),
		EnableBundles:                       configuration.EnableBundles,
		EnableTransitiveRules:               configuration.EnableTransitiveRules,
		EnableAllEventUpload:                configuration.EnableAllEventUpload,
		FullSyncIntervalSeconds:             configuration.FullSyncIntervalSeconds,
		BatchSize:                           configuration.BatchSize,
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
		Description:                         params.Description,
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
		Description:             row.Description,
		Position:                row.Position,
		ClientMode:              ClientMode(row.ClientMode),
		EnableBundles:           row.EnableBundles,
		EnableTransitiveRules:   row.EnableTransitiveRules,
		EnableAllEventUpload:    row.EnableAllEventUpload,
		FullSyncIntervalSeconds: row.FullSyncIntervalSeconds,
		BatchSize:               row.BatchSize,
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
		Targets:         emptyConfigurationTargets(),
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

const configurationSelectSQL = `
SELECT
	c.id,
	c.name,
	c.description,
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
