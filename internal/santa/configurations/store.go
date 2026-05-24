package configurations

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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

var validClientModes = map[ClientMode]bool{
	ClientModeUnknown:    false,
	ClientModeMonitor:    true,
	ClientModeLockdown:   true,
	ClientModeStandalone: true,
}

var validRemovableMediaActions = map[RemovableMediaAction]struct{}{
	RemovableMediaActionAllow:   {},
	RemovableMediaActionBlock:   {},
	RemovableMediaActionRemount: {},
}

func (s *Store) ListConfigurations(
	ctx context.Context,
	params ConfigurationListParams,
) ([]Configuration, int, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
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
	defer rows.Close()

	configurations := []Configuration{}
	configurationIDs := []int64{}
	for rows.Next() {
		configuration, err := scanConfiguration(rows)
		if err != nil {
			return nil, 0, err
		}
		configurations = append(configurations, configuration)
		configurationIDs = append(configurationIDs, configuration.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
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
	configuration := configurationFromSQLC(row)
	return configuration, nil
}

func (s *Store) CreateConfiguration(ctx context.Context, params ConfigurationMutation) (*Configuration, error) {
	cleaned, err := cleanConfigurationMutation(params)
	if err != nil {
		return nil, err
	}

	var configurationID int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		err := validateConfigurationLabelsAvailable(ctx, tx, 0, cleaned.LabelIDs)
		if err != nil {
			return err
		}
		row, err := s.q.WithTx(tx).CreateSantaConfiguration(ctx, createConfigurationParams(cleaned))
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		configurationID = row.ID
		return replaceConfigurationLabels(ctx, tx, configurationID, cleaned.LabelIDs)
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
	cleaned, err := cleanConfigurationMutation(params)
	if err != nil {
		return nil, err
	}

	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		err := validateConfigurationLabelsAvailable(ctx, tx, id, cleaned.LabelIDs)
		if err != nil {
			return err
		}
		row, err := s.q.WithTx(tx).UpdateSantaConfiguration(ctx, updateConfigurationParams(id, cleaned))
		if errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		} else if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		return replaceConfigurationLabels(ctx, tx, row.ID, cleaned.LabelIDs)
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
	cleanedIDs, err := dbutil.ParsePositiveIDs(orderedIDs, "ordered_ids")
	if err != nil {
		return err
	}

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
		if !dbutil.SameIDSet(cleanedIDs, currentIDs) {
			return fmt.Errorf("%w: ordered_ids must exactly match existing configuration IDs", dbutil.ErrInvalidInput)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE santa_configurations c
			SET position = -ordered.position
			FROM unnest($1::bigint[]) WITH ORDINALITY AS ordered(id, position)
			WHERE c.id = ordered.id
		`, cleanedIDs); err != nil {
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
	resolved, err := scanResolvedConfigurationRow(s.db.Pool().QueryRow(ctx, configurationWithMatchedLabelSelectSQL+`
		JOIN santa_configuration_labels cl ON cl.configuration_id = c.id
		JOIN label_membership lm ON lm.label_id = cl.label_id AND lm.host_id = $1
		JOIN labels l ON l.id = cl.label_id
		ORDER BY c.position, c.id, l.name, l.id
		LIMIT 1
	`, hostID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil //nolint:nilnil // no matching configuration is represented by a nil result.
	}
	if err != nil {
		return nil, err
	}
	return resolved, nil
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

func cleanConfigurationMutation(params ConfigurationMutation) (ConfigurationMutation, error) {
	params.Name = strings.TrimSpace(params.Name)
	if params.Name == "" {
		return ConfigurationMutation{}, fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if params.ClientMode == "" {
		params.ClientMode = ClientModeMonitor
	}
	if !validDesiredClientMode(params.ClientMode) {
		return ConfigurationMutation{}, fmt.Errorf("%w: unknown client mode", dbutil.ErrInvalidInput)
	}
	params.AllowedPathRegex = dbutil.CleanStringPtr(params.AllowedPathRegex)
	params.BlockedPathRegex = dbutil.CleanStringPtr(params.BlockedPathRegex)
	params.EventDetailURL = dbutil.CleanStringPtr(params.EventDetailURL)
	params.EventDetailText = dbutil.CleanStringPtr(params.EventDetailText)
	if params.FullSyncIntervalSeconds != nil && *params.FullSyncIntervalSeconds < 60 {
		return ConfigurationMutation{}, fmt.Errorf(
			"%w: full_sync_interval_seconds must be at least 60",
			dbutil.ErrInvalidInput,
		)
	}
	policy, err := cleanRemovableMediaPolicy(params.RemovableMediaPolicy, "removable_media_policy")
	if err != nil {
		return ConfigurationMutation{}, err
	}
	encryptedPolicy, err := cleanRemovableMediaPolicy(
		params.EncryptedRemovableMediaPolicy,
		"encrypted_removable_media_policy",
	)
	if err != nil {
		return ConfigurationMutation{}, err
	}
	params.RemovableMediaPolicy = policy
	params.EncryptedRemovableMediaPolicy = encryptedPolicy
	labelIDs, err := dbutil.CleanPositiveIDList(params.LabelIDs, "label_ids")
	if err != nil {
		return ConfigurationMutation{}, err
	}
	params.LabelIDs = labelIDs
	return params, nil
}

func cleanRemovableMediaPolicy(policy *RemovableMediaPolicy, name string) (*RemovableMediaPolicy, error) {
	if policy == nil {
		return nil, nil //nolint:nilnil // omitted policy is represented by a nil policy and no error.
	}
	cleaned := RemovableMediaPolicy{
		Action:       RemovableMediaAction(strings.TrimSpace(string(policy.Action))),
		RemountFlags: cleanStringList(policy.RemountFlags),
	}
	if cleaned.Action == "" {
		return nil, nil //nolint:nilnil // an empty policy object clears the optional policy.
	}
	if !validRemovableMediaAction(cleaned.Action) {
		return nil, fmt.Errorf("%w: unknown %s action", dbutil.ErrInvalidInput, name)
	}
	if cleaned.Action == RemovableMediaActionRemount && len(cleaned.RemountFlags) == 0 {
		return nil, fmt.Errorf("%w: %s.remount_flags are required when action is remount", dbutil.ErrInvalidInput, name)
	}
	return &cleaned, nil
}

func cleanStringList(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	slices.Sort(cleaned)
	return slices.Compact(cleaned)
}

func validDesiredClientMode(clientMode ClientMode) bool {
	return validClientModes[clientMode]
}

func validRemovableMediaAction(action RemovableMediaAction) bool {
	_, ok := validRemovableMediaActions[action]
	return ok
}

func configurationListWhere(params ConfigurationListParams) (string, []any) {
	var where dbutil.WhereBuilder
	if params.Q != "" {
		search := where.Arg("%" + params.Q + "%")
		where.Add("c.name ILIKE " + search)
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
		FullSyncIntervalSeconds:             int32Ptr(configuration.FullSyncIntervalSeconds),
		BatchSize:                           int32Ptr(configuration.BatchSize),
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
		FullSyncIntervalSeconds: intPtrFromSQLC(row.FullSyncIntervalSeconds),
		BatchSize:               intPtrFromSQLC(row.BatchSize),
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

func int32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

func intPtrFromSQLC(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func removableMediaPolicySQLC(
	policy *RemovableMediaPolicy,
) (*sqlc.SantaRemovableMediaAction, []string) {
	if policy == nil {
		return nil, nil
	}
	action := sqlc.SantaRemovableMediaAction(policy.Action)
	return &action, nilIfEmptyStrings(policy.RemountFlags)
}

func removableMediaPolicyFromSQLC(
	action *sqlc.SantaRemovableMediaAction,
	flags []string,
) *RemovableMediaPolicy {
	if action == nil {
		return nil
	}
	return &RemovableMediaPolicy{
		Action:       RemovableMediaAction(*action),
		RemountFlags: flags,
	}
}

func scanConfiguration(row pgx.Row) (Configuration, error) {
	configuration, err := scanConfigurationRow(row)
	if err != nil {
		return Configuration{}, err
	}
	return *configuration, nil
}

func scanConfigurationRow(row pgx.Row) (*Configuration, error) {
	var configuration Configuration
	var clientMode string
	var fullSyncInterval pgtype.Int4
	var batchSize pgtype.Int4
	var enableBundles pgtype.Bool
	var enableTransitiveRules pgtype.Bool
	var enableAllEventUpload pgtype.Bool
	var allowedPathRegex pgtype.Text
	var blockedPathRegex pgtype.Text
	var removableMediaAction pgtype.Text
	var removableMediaRemountFlags []string
	var encryptedRemovableMediaAction pgtype.Text
	var encryptedRemovableMediaRemountFlags []string
	var eventDetailURL pgtype.Text
	var eventDetailText pgtype.Text

	err := row.Scan(
		&configuration.ID,
		&configuration.Name,
		&configuration.Position,
		&clientMode,
		&enableBundles,
		&enableTransitiveRules,
		&enableAllEventUpload,
		&fullSyncInterval,
		&batchSize,
		&allowedPathRegex,
		&blockedPathRegex,
		&removableMediaAction,
		&removableMediaRemountFlags,
		&encryptedRemovableMediaAction,
		&encryptedRemovableMediaRemountFlags,
		&eventDetailURL,
		&eventDetailText,
		&configuration.CreatedAt,
		&configuration.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	hydrateConfiguration(
		&configuration,
		clientMode,
		enableBundles,
		enableTransitiveRules,
		enableAllEventUpload,
		fullSyncInterval,
		batchSize,
		allowedPathRegex,
		blockedPathRegex,
		removableMediaAction,
		removableMediaRemountFlags,
		encryptedRemovableMediaAction,
		encryptedRemovableMediaRemountFlags,
		eventDetailURL,
		eventDetailText,
	)
	return &configuration, nil
}

func scanResolvedConfigurationRow(row pgx.Row) (*ResolvedConfiguration, error) {
	configuration, label, err := scanConfigurationAndMatchedLabel(row)
	if err != nil {
		return nil, err
	}
	return &ResolvedConfiguration{Configuration: *configuration, MatchedViaLabel: label}, nil
}

func scanConfigurationAndMatchedLabel(row pgx.Row) (*Configuration, *LabelMatch, error) {
	var configuration Configuration
	var label LabelMatch
	var clientMode string
	var fullSyncInterval pgtype.Int4
	var batchSize pgtype.Int4
	var enableBundles pgtype.Bool
	var enableTransitiveRules pgtype.Bool
	var enableAllEventUpload pgtype.Bool
	var allowedPathRegex pgtype.Text
	var blockedPathRegex pgtype.Text
	var removableMediaAction pgtype.Text
	var removableMediaRemountFlags []string
	var encryptedRemovableMediaAction pgtype.Text
	var encryptedRemovableMediaRemountFlags []string
	var eventDetailURL pgtype.Text
	var eventDetailText pgtype.Text

	err := row.Scan(
		&configuration.ID,
		&configuration.Name,
		&configuration.Position,
		&clientMode,
		&enableBundles,
		&enableTransitiveRules,
		&enableAllEventUpload,
		&fullSyncInterval,
		&batchSize,
		&allowedPathRegex,
		&blockedPathRegex,
		&removableMediaAction,
		&removableMediaRemountFlags,
		&encryptedRemovableMediaAction,
		&encryptedRemovableMediaRemountFlags,
		&eventDetailURL,
		&eventDetailText,
		&configuration.CreatedAt,
		&configuration.UpdatedAt,
		&label.ID,
		&label.Name,
	)
	if err != nil {
		return nil, nil, err
	}
	hydrateConfiguration(
		&configuration,
		clientMode,
		enableBundles,
		enableTransitiveRules,
		enableAllEventUpload,
		fullSyncInterval,
		batchSize,
		allowedPathRegex,
		blockedPathRegex,
		removableMediaAction,
		removableMediaRemountFlags,
		encryptedRemovableMediaAction,
		encryptedRemovableMediaRemountFlags,
		eventDetailURL,
		eventDetailText,
	)
	return &configuration, &label, nil
}

func hydrateConfiguration(
	configuration *Configuration,
	clientMode string,
	enableBundles pgtype.Bool,
	enableTransitiveRules pgtype.Bool,
	enableAllEventUpload pgtype.Bool,
	fullSyncInterval pgtype.Int4,
	batchSize pgtype.Int4,
	allowedPathRegex pgtype.Text,
	blockedPathRegex pgtype.Text,
	removableMediaAction pgtype.Text,
	removableMediaRemountFlags []string,
	encryptedRemovableMediaAction pgtype.Text,
	encryptedRemovableMediaRemountFlags []string,
	eventDetailURL pgtype.Text,
	eventDetailText pgtype.Text,
) {
	configuration.ClientMode = ClientMode(clientMode)
	configuration.EnableBundles = boolPtrFromPG(enableBundles)
	configuration.EnableTransitiveRules = boolPtrFromPG(enableTransitiveRules)
	configuration.EnableAllEventUpload = boolPtrFromPG(enableAllEventUpload)
	configuration.FullSyncIntervalSeconds = intPtrFromPG(fullSyncInterval)
	configuration.BatchSize = intPtrFromPG(batchSize)
	configuration.AllowedPathRegex = stringPtrFromPG(allowedPathRegex)
	configuration.BlockedPathRegex = stringPtrFromPG(blockedPathRegex)
	configuration.RemovableMediaPolicy = removableMediaPolicyFromPG(
		removableMediaAction,
		removableMediaRemountFlags,
	)
	configuration.EncryptedRemovableMediaPolicy = removableMediaPolicyFromPG(
		encryptedRemovableMediaAction,
		encryptedRemovableMediaRemountFlags,
	)
	configuration.EventDetailURL = stringPtrFromPG(eventDetailURL)
	configuration.EventDetailText = stringPtrFromPG(eventDetailText)
}

func boolPtrFromPG(value pgtype.Bool) *bool {
	if !value.Valid {
		return nil
	}
	return &value.Bool
}

func intPtrFromPG(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	out := int(value.Int32)
	return &out
}

func stringPtrFromPG(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func removableMediaPolicyFromPG(value pgtype.Text, flags []string) *RemovableMediaPolicy {
	if !value.Valid {
		return nil
	}
	return &RemovableMediaPolicy{
		Action:       RemovableMediaAction(value.String),
		RemountFlags: flags,
	}
}

func nilIfEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return values
}

const configurationSelectFields = `
	c.id,
	c.name,
	c.position,
	c.client_mode::text,
	c.enable_bundles,
	c.enable_transitive_rules,
	c.enable_all_event_upload,
	c.full_sync_interval_seconds,
	c.batch_size,
	c.allowed_path_regex,
	c.blocked_path_regex,
	c.removable_media_action::text,
	c.removable_media_remount_flags,
	c.encrypted_removable_media_action::text,
	c.encrypted_removable_media_remount_flags,
	c.event_detail_url,
	c.event_detail_text,
	c.created_at,
	c.updated_at`

const configurationSelectSQL = `
SELECT` + configurationSelectFields + `
FROM santa_configurations c`

const configurationWithMatchedLabelSelectSQL = `
SELECT` + configurationSelectFields + `,
	l.id,
	l.name
FROM santa_configurations c`
