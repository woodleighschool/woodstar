package santa

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/database"
	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// Store persists Santa state.
type Store struct {
	db *database.DB
}

func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

func (s *Store) UpsertHostObservation(ctx context.Context, observation HostObservation) error {
	observation.MachineID = strings.TrimSpace(observation.MachineID)
	observation.SerialNumber = strings.TrimSpace(observation.SerialNumber)
	if observation.HostID <= 0 || observation.MachineID == "" || observation.SerialNumber == "" {
		return dbutil.ErrInvalidInput
	}
	if observation.ClientModeReported == "" {
		observation.ClientModeReported = ClientModeUnknown
	}
	if observation.PrimaryUserGroups == nil {
		observation.PrimaryUserGroups = []string{}
	}

	_, err := s.db.Pool().Exec(ctx, `
		INSERT INTO santa_hosts (
			host_id,
			machine_id,
			serial_number,
			santa_version,
			client_mode_reported,
			primary_user,
			primary_user_groups,
			sip_status,
			os_build,
			model_identifier,
			last_seen_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, COALESCE($11, now()))
		ON CONFLICT (host_id) DO UPDATE SET
			machine_id = EXCLUDED.machine_id,
			serial_number = EXCLUDED.serial_number,
			santa_version = EXCLUDED.santa_version,
			client_mode_reported = EXCLUDED.client_mode_reported,
			primary_user = EXCLUDED.primary_user,
			primary_user_groups = EXCLUDED.primary_user_groups,
			sip_status = EXCLUDED.sip_status,
			os_build = EXCLUDED.os_build,
			model_identifier = EXCLUDED.model_identifier,
			last_seen_at = EXCLUDED.last_seen_at,
			updated_at = now()
	`, observation.HostID,
		observation.MachineID,
		observation.SerialNumber,
		observation.Version,
		observation.ClientModeReported,
		observation.PrimaryUser,
		observation.PrimaryUserGroups,
		observation.SIPStatus,
		observation.OSBuild,
		observation.ModelIdentifier,
		observation.LastSeenAt,
	)
	return err
}

func (s *Store) LoadHostState(ctx context.Context, hostID int64) (*HostState, error) {
	var detail HostState
	var clientMode string
	var desiredPayload []byte
	var appliedPayload []byte
	err := s.db.Pool().QueryRow(ctx, `
		SELECT
			sh.santa_version,
			sh.client_mode_reported::text,
			sh.last_seen_at,
			COALESCE(ss.desired_targets, '[]'::jsonb),
			COALESCE(ss.applied_targets, '[]'::jsonb),
			ss.last_clean_sync_at
		FROM santa_hosts sh
		LEFT JOIN santa_sync_state ss ON ss.host_id = sh.host_id
		WHERE sh.host_id = $1
	`, hostID).Scan(
		&detail.Version,
		&clientMode,
		&detail.LastSyncAt,
		&desiredPayload,
		&appliedPayload,
		&detail.RuleSync.LastCleanSyncAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil //nolint:nilnil // missing Santa observation is represented by a nil state.
	}
	if err != nil {
		return nil, err
	}

	ruleSync, err := syncSummary(desiredPayload, appliedPayload)
	if err != nil {
		return nil, err
	}
	ruleSync.LastCleanSyncAt = detail.RuleSync.LastCleanSyncAt

	detail.Enrolled = true
	detail.ClientModeReported = ClientMode(clientMode)
	detail.RuleSync = ruleSync
	effectiveConfiguration, err := s.ResolveConfigurationForHost(ctx, hostID)
	if err != nil {
		return nil, err
	}
	if effectiveConfiguration != nil {
		detail.EffectiveConfiguration = &EffectiveConfiguration{
			ID:              effectiveConfiguration.ID,
			Name:            effectiveConfiguration.Name,
			MatchedViaLabel: effectiveConfiguration.MatchedViaLabel,
		}
	}
	return &detail, nil
}

func (s *Store) ListSyncTokens(ctx context.Context) ([]SyncToken, error) {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT id, value_hash, created_at, last_used_at
		FROM santa_sync_tokens
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := []SyncToken{}
	for rows.Next() {
		var token SyncToken
		if err := rows.Scan(&token.ID, &token.ValueHash, &token.CreatedAt, &token.LastUsedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (s *Store) CreateSyncToken(ctx context.Context) (CreatedSyncToken, error) {
	value, err := randomToken()
	if err != nil {
		return CreatedSyncToken{}, err
	}

	token := CreatedSyncToken{Value: value}
	err = s.db.Pool().QueryRow(ctx, `
		INSERT INTO santa_sync_tokens (value_hash)
		VALUES ($1)
		RETURNING id, value_hash, created_at, last_used_at
	`, hashToken(value)).Scan(
		&token.ID,
		&token.ValueHash,
		&token.CreatedAt,
		&token.LastUsedAt,
	)
	if err != nil {
		return CreatedSyncToken{}, err
	}
	return token, nil
}

func (s *Store) DeleteSyncToken(ctx context.Context, id int64) error {
	if id <= 0 {
		return dbutil.ErrNotFound
	}

	var deletedID int64
	err := s.db.Pool().QueryRow(ctx, `
		DELETE FROM santa_sync_tokens
		WHERE id = $1
		RETURNING id
	`, id).Scan(&deletedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return dbutil.ErrNotFound
	}
	return err
}

func (s *Store) VerifyBearerToken(ctx context.Context, authorization string) (bool, error) {
	value, ok := parseBearerToken(authorization)
	if !ok {
		return false, nil
	}

	valueHash := hashToken(value)
	var exists bool
	if err := s.db.Pool().QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM santa_sync_tokens
			WHERE value_hash = $1
		)
	`, valueHash).Scan(&exists); err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	_, _ = s.db.Pool().Exec(ctx, `
		UPDATE santa_sync_tokens
		SET last_used_at = now()
		WHERE value_hash = $1
	`, valueHash)
	return true, nil
}

func parseBearerToken(authorization string) (string, bool) {
	scheme, value, ok := strings.Cut(authorization, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	value = strings.TrimSpace(value)
	return value, value != "" && !strings.Contains(value, " ")
}

func hashToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

type syncTargetFingerprint struct {
	RuleType    string `json:"rule_type"`
	Identifier  string `json:"identifier"`
	PayloadHash string `json:"payload_hash"`
}

func (target syncTargetFingerprint) key() string {
	return target.RuleType + "\x00" + target.Identifier + "\x00" + target.PayloadHash
}

func syncSummary(desiredPayload []byte, appliedPayload []byte) (RuleSyncSummary, error) {
	desired, err := decodeSyncTargets(desiredPayload)
	if err != nil {
		return RuleSyncSummary{}, err
	}
	applied, err := decodeSyncTargets(appliedPayload)
	if err != nil {
		return RuleSyncSummary{}, err
	}
	return RuleSyncSummary{
		DesiredCount: len(desired),
		AppliedCount: len(applied),
		PendingCount: pendingSyncTargetCount(desired, applied),
	}, nil
}

func decodeSyncTargets(payload []byte) ([]syncTargetFingerprint, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	var targets []syncTargetFingerprint
	if err := json.Unmarshal(payload, &targets); err != nil {
		return nil, err
	}
	return targets, nil
}

func pendingSyncTargetCount(desired []syncTargetFingerprint, applied []syncTargetFingerprint) int {
	desiredSet := syncTargetSet(desired)
	appliedSet := syncTargetSet(applied)

	var pending int
	for key := range desiredSet {
		if !appliedSet[key] {
			pending++
		}
	}
	for key := range appliedSet {
		if !desiredSet[key] {
			pending++
		}
	}
	return pending
}

func syncTargetSet(targets []syncTargetFingerprint) map[string]bool {
	out := make(map[string]bool, len(targets))
	for _, target := range targets {
		out[target.key()] = true
	}
	return out
}
