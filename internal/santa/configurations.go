package santa

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

type RemovableMediaAction string

const (
	RemovableMediaActionAllow   RemovableMediaAction = "allow"
	RemovableMediaActionBlock   RemovableMediaAction = "block"
	RemovableMediaActionRemount RemovableMediaAction = "remount"
)

type ConfigurationListParams struct {
	dbutil.ListParams
}

type ConfigurationCreate struct {
	Name                                string                `json:"name"`
	ClientMode                          ClientMode            `json:"client_mode,omitempty"`
	EnableBundles                       *bool                 `json:"enable_bundles,omitempty"`
	EnableTransitiveRules               *bool                 `json:"enable_transitive_rules,omitempty"`
	EnableAllEventUpload                *bool                 `json:"enable_all_event_upload,omitempty"`
	FullSyncIntervalSeconds             *int                  `json:"full_sync_interval_seconds,omitempty"`
	BatchSize                           *int                  `json:"batch_size,omitempty"`
	AllowedPathRegex                    *string               `json:"allowed_path_regex,omitempty"`
	BlockedPathRegex                    *string               `json:"blocked_path_regex,omitempty"`
	RemovableMediaAction                *RemovableMediaAction `json:"removable_media_action,omitempty"`
	RemovableMediaRemountFlags          []string              `json:"removable_media_remount_flags,omitempty"`
	EncryptedRemovableMediaAction       *RemovableMediaAction `json:"encrypted_removable_media_action,omitempty"`
	EncryptedRemovableMediaRemountFlags []string              `json:"encrypted_removable_media_remount_flags,omitempty"`
	EventDetailURL                      *string               `json:"event_detail_url,omitempty"`
	EventDetailText                     *string               `json:"event_detail_text,omitempty"`
	LabelIDs                            []int64               `json:"label_ids,omitempty"`
}

type ConfigurationUpdate ConfigurationCreate

type Configuration struct {
	ID                                  int64                 `json:"id"`
	Name                                string                `json:"name"`
	Position                            int                   `json:"position"`
	ClientMode                          ClientMode            `json:"client_mode"`
	EnableBundles                       *bool                 `json:"enable_bundles,omitempty"`
	EnableTransitiveRules               *bool                 `json:"enable_transitive_rules,omitempty"`
	EnableAllEventUpload                *bool                 `json:"enable_all_event_upload,omitempty"`
	FullSyncIntervalSeconds             *int                  `json:"full_sync_interval_seconds,omitempty"`
	BatchSize                           *int                  `json:"batch_size,omitempty"`
	AllowedPathRegex                    *string               `json:"allowed_path_regex,omitempty"`
	BlockedPathRegex                    *string               `json:"blocked_path_regex,omitempty"`
	RemovableMediaAction                *RemovableMediaAction `json:"removable_media_action,omitempty"`
	RemovableMediaRemountFlags          []string              `json:"removable_media_remount_flags,omitempty"`
	EncryptedRemovableMediaAction       *RemovableMediaAction `json:"encrypted_removable_media_action,omitempty"`
	EncryptedRemovableMediaRemountFlags []string              `json:"encrypted_removable_media_remount_flags,omitempty"`
	EventDetailURL                      *string               `json:"event_detail_url,omitempty"`
	EventDetailText                     *string               `json:"event_detail_text,omitempty"`
	LabelIDs                            []int64               `json:"label_ids"`
	CreatedAt                           time.Time             `json:"created_at"`
	UpdatedAt                           time.Time             `json:"updated_at"`
}

type ResolvedConfiguration struct {
	Configuration
	MatchedViaLabel *MatchedLabel `json:"matched_via_label,omitempty"`
}

type ConfigurationLabelConflict struct {
	Code              string `json:"code"`
	LabelID           int64  `json:"label_id"`
	ConfigurationID   int64  `json:"configuration_id"`
	ConfigurationName string `json:"configuration_name"`
}

func (e *ConfigurationLabelConflict) Error() string {
	return "configuration label already belongs to another configuration"
}

func (s *Store) ListConfigurations(
	ctx context.Context,
	params ConfigurationListParams,
) ([]Configuration, int, error) {
	params.ListParams = dbutil.CleanListParams(params.ListParams)
	where, args := configurationListWhere(params)

	var count int
	if err := s.db.Pool().QueryRow(ctx, "SELECT count(*) FROM santa_configurations c "+where, args...).Scan(&count); err != nil {
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
	configuration, err := scanConfigurationRow(s.db.Pool().QueryRow(ctx, configurationSelectSQL+" WHERE c.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dbutil.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return configuration, nil
}

func (s *Store) CreateConfiguration(ctx context.Context, params ConfigurationCreate) (*Configuration, error) {
	cleaned, err := cleanConfigurationCreate(params)
	if err != nil {
		return nil, err
	}

	var configurationID int64
	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		err := validateConfigurationLabelsAvailable(ctx, tx, 0, cleaned.LabelIDs)
		if err != nil {
			return err
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO santa_configurations (
				name,
				position,
				client_mode,
				enable_bundles,
				enable_transitive_rules,
				enable_all_event_upload,
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
			)
			VALUES (
				$1,
				(SELECT COALESCE(MAX(position) + 1, 0) FROM santa_configurations),
				$2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
			)
			RETURNING id
		`, configurationInsertArgs(cleaned)...).Scan(&configurationID)
		if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
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
	params ConfigurationUpdate,
) (*Configuration, error) {
	cleaned, err := cleanConfigurationUpdate(params)
	if err != nil {
		return nil, err
	}

	err = s.db.WithTx(ctx, func(tx pgx.Tx) error {
		err := validateConfigurationLabelsAvailable(ctx, tx, id, cleaned.LabelIDs)
		if err != nil {
			return err
		}
		row := tx.QueryRow(ctx, `
			UPDATE santa_configurations
			SET
				name = $1,
				client_mode = $2,
				enable_bundles = $3,
				enable_transitive_rules = $4,
				enable_all_event_upload = $5,
				full_sync_interval_seconds = $6,
				batch_size = $7,
				allowed_path_regex = $8,
				blocked_path_regex = $9,
				removable_media_action = $10,
				removable_media_remount_flags = $11,
				encrypted_removable_media_action = $12,
				encrypted_removable_media_remount_flags = $13,
				event_detail_url = $14,
				event_detail_text = $15,
				updated_at = now()
			WHERE id = $16
			RETURNING id
		`, append(configurationInsertArgs(ConfigurationCreate(cleaned)), id)...)
		var updatedID int64
		if err := row.Scan(&updatedID); errors.Is(err, pgx.ErrNoRows) {
			return dbutil.ErrNotFound
		} else if err != nil {
			if dbutil.IsUniqueViolation(err) {
				return dbutil.ErrAlreadyExists
			}
			return err
		}
		return replaceConfigurationLabels(ctx, tx, updatedID, cleaned.LabelIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.GetConfigurationByID(ctx, id)
}

func (s *Store) DeleteConfiguration(ctx context.Context, id int64) error {
	var deletedID int64
	err := s.db.Pool().QueryRow(ctx, `
		DELETE FROM santa_configurations
		WHERE id = $1
		RETURNING id
	`, id).Scan(&deletedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func (s *Store) ReorderConfigurations(ctx context.Context, orderedIDs []int64) error {
	ids, err := parsePositiveIDs(orderedIDs, "ordered_ids")
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
		if !sameIDSet(ids, currentIDs) {
			return fmt.Errorf("%w: ordered_ids must exactly match existing configuration IDs", dbutil.ErrInvalidInput)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE santa_configurations
			SET position = position + 100000
		`); err != nil {
			return err
		}
		for position, id := range ids {
			if _, err := tx.Exec(ctx, `
				UPDATE santa_configurations
				SET position = $1
				WHERE id = $2
			`, position, id); err != nil {
				return err
			}
		}
		return nil
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
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

func replaceConfigurationLabels(ctx context.Context, tx pgx.Tx, configurationID int64, labelIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM santa_configuration_labels WHERE configuration_id = $1`, configurationID); err != nil {
		return err
	}
	for _, labelID := range labelIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO santa_configuration_labels (label_id, configuration_id)
			VALUES ($1, $2)
		`, labelID, configurationID); err != nil {
			return err
		}
	}
	return nil
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

	var conflict ConfigurationLabelConflict
	err := tx.QueryRow(ctx, `
		SELECT
			'configuration_label_conflict',
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
		&conflict.Code,
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

func cleanConfigurationCreate(params ConfigurationCreate) (ConfigurationCreate, error) {
	params.Name = strings.TrimSpace(params.Name)
	if params.Name == "" {
		return ConfigurationCreate{}, fmt.Errorf("%w: name is required", dbutil.ErrInvalidInput)
	}
	if params.ClientMode == "" {
		params.ClientMode = ClientModeMonitor
	}
	if !validDesiredClientMode(params.ClientMode) {
		return ConfigurationCreate{}, fmt.Errorf("%w: unknown client mode", dbutil.ErrInvalidInput)
	}
	params.AllowedPathRegex = dbutil.CleanStringPtr(params.AllowedPathRegex)
	params.BlockedPathRegex = dbutil.CleanStringPtr(params.BlockedPathRegex)
	params.EventDetailURL = dbutil.CleanStringPtr(params.EventDetailURL)
	params.EventDetailText = dbutil.CleanStringPtr(params.EventDetailText)
	if params.FullSyncIntervalSeconds != nil && *params.FullSyncIntervalSeconds < 60 {
		return ConfigurationCreate{}, fmt.Errorf("%w: full_sync_interval_seconds must be at least 60", dbutil.ErrInvalidInput)
	}
	params.RemovableMediaAction = cleanRemovableMediaActionPtr(params.RemovableMediaAction)
	params.EncryptedRemovableMediaAction = cleanRemovableMediaActionPtr(params.EncryptedRemovableMediaAction)
	params.RemovableMediaRemountFlags = cleanStringList(params.RemovableMediaRemountFlags)
	params.EncryptedRemovableMediaRemountFlags = cleanStringList(params.EncryptedRemovableMediaRemountFlags)
	if err := validateRemountFlags(params.RemovableMediaAction, params.RemovableMediaRemountFlags, "removable_media_remount_flags"); err != nil {
		return ConfigurationCreate{}, err
	}
	if err := validateRemountFlags(
		params.EncryptedRemovableMediaAction,
		params.EncryptedRemovableMediaRemountFlags,
		"encrypted_removable_media_remount_flags",
	); err != nil {
		return ConfigurationCreate{}, err
	}
	labelIDs, err := cleanLabelIDs(params.LabelIDs, "label_ids")
	if err != nil {
		return ConfigurationCreate{}, err
	}
	params.LabelIDs = labelIDs
	return params, nil
}

func cleanConfigurationUpdate(params ConfigurationUpdate) (ConfigurationUpdate, error) {
	cleaned, err := cleanConfigurationCreate(ConfigurationCreate(params))
	return ConfigurationUpdate(cleaned), err
}

func cleanRemovableMediaActionPtr(value *RemovableMediaAction) *RemovableMediaAction {
	if value == nil {
		return nil
	}
	cleaned := RemovableMediaAction(strings.TrimSpace(string(*value)))
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

func validateRemountFlags(action *RemovableMediaAction, flags []string, name string) error {
	if action == nil {
		return nil
	}
	if !validRemovableMediaAction(*action) {
		return fmt.Errorf("%w: unknown removable media action", dbutil.ErrInvalidInput)
	}
	if *action == RemovableMediaActionRemount && len(flags) == 0 {
		return fmt.Errorf("%w: %s are required when action is remount", dbutil.ErrInvalidInput, name)
	}
	return nil
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
	switch clientMode {
	case ClientModeMonitor, ClientModeLockdown, ClientModeStandalone:
		return true
	default:
		return false
	}
}

func validRemovableMediaAction(action RemovableMediaAction) bool {
	switch action {
	case RemovableMediaActionAllow, RemovableMediaActionBlock, RemovableMediaActionRemount:
		return true
	default:
		return false
	}
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

func configurationInsertArgs(configuration ConfigurationCreate) []any {
	return []any{
		configuration.Name,
		configuration.ClientMode,
		configuration.EnableBundles,
		configuration.EnableTransitiveRules,
		configuration.EnableAllEventUpload,
		configuration.FullSyncIntervalSeconds,
		configuration.BatchSize,
		configuration.AllowedPathRegex,
		configuration.BlockedPathRegex,
		configuration.RemovableMediaAction,
		nilIfEmptyStrings(configuration.RemovableMediaRemountFlags),
		configuration.EncryptedRemovableMediaAction,
		nilIfEmptyStrings(configuration.EncryptedRemovableMediaRemountFlags),
		configuration.EventDetailURL,
		configuration.EventDetailText,
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
	var encryptedRemovableMediaAction pgtype.Text
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
		&configuration.RemovableMediaRemountFlags,
		&encryptedRemovableMediaAction,
		&configuration.EncryptedRemovableMediaRemountFlags,
		&eventDetailURL,
		&eventDetailText,
		&configuration.CreatedAt,
		&configuration.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	hydrateConfiguration(&configuration, clientMode, enableBundles, enableTransitiveRules, enableAllEventUpload, fullSyncInterval, batchSize, allowedPathRegex, blockedPathRegex, removableMediaAction, encryptedRemovableMediaAction, eventDetailURL, eventDetailText)
	return &configuration, nil
}

func scanResolvedConfigurationRow(row pgx.Row) (*ResolvedConfiguration, error) {
	configuration, label, err := scanConfigurationAndMatchedLabel(row)
	if err != nil {
		return nil, err
	}
	return &ResolvedConfiguration{Configuration: *configuration, MatchedViaLabel: label}, nil
}

func scanConfigurationAndMatchedLabel(row pgx.Row) (*Configuration, *MatchedLabel, error) {
	var configuration Configuration
	var label MatchedLabel
	var clientMode string
	var fullSyncInterval pgtype.Int4
	var batchSize pgtype.Int4
	var enableBundles pgtype.Bool
	var enableTransitiveRules pgtype.Bool
	var enableAllEventUpload pgtype.Bool
	var allowedPathRegex pgtype.Text
	var blockedPathRegex pgtype.Text
	var removableMediaAction pgtype.Text
	var encryptedRemovableMediaAction pgtype.Text
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
		&configuration.RemovableMediaRemountFlags,
		&encryptedRemovableMediaAction,
		&configuration.EncryptedRemovableMediaRemountFlags,
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
	hydrateConfiguration(&configuration, clientMode, enableBundles, enableTransitiveRules, enableAllEventUpload, fullSyncInterval, batchSize, allowedPathRegex, blockedPathRegex, removableMediaAction, encryptedRemovableMediaAction, eventDetailURL, eventDetailText)
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
	encryptedRemovableMediaAction pgtype.Text,
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
	configuration.RemovableMediaAction = removableMediaActionPtrFromPG(removableMediaAction)
	configuration.EncryptedRemovableMediaAction = removableMediaActionPtrFromPG(encryptedRemovableMediaAction)
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

func removableMediaActionPtrFromPG(value pgtype.Text) *RemovableMediaAction {
	if !value.Valid {
		return nil
	}
	action := RemovableMediaAction(value.String)
	return &action
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
