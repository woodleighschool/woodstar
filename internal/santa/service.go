package santa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

// ErrNotImplemented indicates a sync stage route exists before its service
// behavior has landed.
var ErrNotImplemented = errors.New("santa sync stage not implemented")

// Service coordinates Santa sync protocol stages.
type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) HandlePreflight(
	ctx context.Context,
	machineID string,
	req *syncv1.PreflightRequest,
) (*syncv1.PreflightResponse, error) {
	hostID, err := s.store.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return nil, err
	}
	if err := s.store.UpsertHostObservation(ctx, hostObservationFromPreflight(hostID, machineID, req)); err != nil {
		return nil, err
	}

	effectiveRules, err := s.store.ResolveRulesForHost(ctx, hostID)
	if err != nil {
		return nil, err
	}
	targets := syncTargetsFromRules(effectiveRules)
	pending := targets
	syncType := syncv1.SyncType_NORMAL
	pendingFullSync := false
	if req.GetRequestCleanSync() {
		syncType = syncv1.SyncType_CLEAN
		pendingFullSync = true
	}
	if err := s.store.replacePendingSync(ctx, hostID, req.GetRulesHash(), targets, pending, pendingFullSync); err != nil {
		return nil, err
	}

	resp := &syncv1.PreflightResponse{SyncType: &syncType}
	configuration, err := s.store.ResolveConfigurationForHost(ctx, hostID)
	if err != nil {
		return nil, err
	}
	if configuration != nil {
		applyConfigurationToPreflightResponse(resp, &configuration.Configuration)
	}
	return resp, nil
}

func (s *Service) HandleEventUpload(
	context.Context,
	string,
	*syncv1.EventUploadRequest,
) (*syncv1.EventUploadResponse, error) {
	return &syncv1.EventUploadResponse{}, nil
}

func (s *Service) HandleRuleDownload(
	ctx context.Context,
	machineID string,
	_ *syncv1.RuleDownloadRequest,
) (*syncv1.RuleDownloadResponse, error) {
	hostID, err := s.store.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return nil, err
	}
	targets, err := s.store.loadPendingSyncTargets(ctx, hostID)
	if err != nil {
		return nil, err
	}
	return &syncv1.RuleDownloadResponse{Rules: protoRulesFromSyncTargets(targets)}, nil
}

func (s *Service) HandlePostflight(
	ctx context.Context,
	machineID string,
	req *syncv1.PostflightRequest,
) (*syncv1.PostflightResponse, error) {
	hostID, err := s.store.hostIDByMachineID(ctx, machineID)
	if err != nil {
		return nil, err
	}
	if err := s.store.promotePendingSync(ctx, hostID, req.GetRulesHash(), int(req.GetRulesReceived()), int(req.GetRulesProcessed())); err != nil {
		return nil, err
	}
	return &syncv1.PostflightResponse{}, nil
}

type syncTarget struct {
	RuleType      string `json:"rule_type"`
	Identifier    string `json:"identifier"`
	Policy        string `json:"policy"`
	CELExpression string `json:"cel_expression,omitempty"`
	CustomMessage string `json:"custom_message,omitempty"`
	CustomURL     string `json:"custom_url,omitempty"`
	PayloadHash   string `json:"payload_hash"`
}

func (s *Store) hostIDByMachineID(ctx context.Context, machineID string) (int64, error) {
	machineID = strings.TrimSpace(machineID)
	if machineID == "" {
		return 0, dbutil.ErrNotFound
	}
	var hostID int64
	err := s.db.Pool().QueryRow(ctx, `
		SELECT id
		FROM hosts
		WHERE hardware_uuid = $1
			AND deleted_at IS NULL
	`, machineID).Scan(&hostID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, dbutil.ErrNotFound
	}
	return hostID, err
}

func hostObservationFromPreflight(hostID int64, machineID string, req *syncv1.PreflightRequest) HostObservation {
	var sipStatus *int16
	if req.GetSipStatus() != 0 {
		value := int16(req.GetSipStatus())
		sipStatus = &value
	}
	return HostObservation{
		HostID:             hostID,
		MachineID:          machineID,
		SerialNumber:       req.GetSerialNumber(),
		Version:            req.GetSantaVersion(),
		ClientModeReported: clientModeFromProto(req.GetClientMode()),
		PrimaryUser:        req.GetPrimaryUser(),
		PrimaryUserGroups:  req.GetPrimaryUserGroups(),
		SIPStatus:          sipStatus,
		OSBuild:            req.GetOsBuild(),
		ModelIdentifier:    req.GetModelIdentifier(),
	}
}

func syncTargetsFromRules(rules []EffectiveRule) []syncTarget {
	targets := make([]syncTarget, 0, len(rules))
	for _, rule := range rules {
		target := syncTarget{
			RuleType:      string(rule.RuleType),
			Identifier:    rule.Identifier,
			Policy:        string(rule.Policy),
			CELExpression: rule.CELExpression,
			CustomMessage: rule.CustomMessage,
			CustomURL:     rule.CustomURL,
		}
		target.PayloadHash = syncTargetPayloadHash(target)
		targets = append(targets, target)
	}
	return targets
}

func syncTargetPayloadHash(target syncTarget) string {
	target.PayloadHash = ""
	payload, _ := json.Marshal(target)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (s *Store) replacePendingSync(
	ctx context.Context,
	hostID int64,
	clientRulesHash string,
	desired []syncTarget,
	pending []syncTarget,
	pendingFullSync bool,
) error {
	desiredPayload, err := json.Marshal(desired)
	if err != nil {
		return err
	}
	pendingPayload, err := json.Marshal(pending)
	if err != nil {
		return err
	}
	_, err = s.db.Pool().Exec(ctx, `
		INSERT INTO santa_sync_state (
			host_id,
			client_rules_hash,
			desired_targets,
			pending_payload,
			pending_payload_rule_count,
			pending_full_sync,
			pending_preflight_at,
			last_rule_sync_attempt_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, now(), now(), now())
		ON CONFLICT (host_id) DO UPDATE SET
			client_rules_hash = EXCLUDED.client_rules_hash,
			desired_targets = EXCLUDED.desired_targets,
			pending_payload = EXCLUDED.pending_payload,
			pending_payload_rule_count = EXCLUDED.pending_payload_rule_count,
			pending_full_sync = EXCLUDED.pending_full_sync,
			pending_preflight_at = EXCLUDED.pending_preflight_at,
			last_rule_sync_attempt_at = EXCLUDED.last_rule_sync_attempt_at,
			updated_at = now()
	`, hostID, clientRulesHash, desiredPayload, pendingPayload, len(pending), pendingFullSync)
	return err
}

func (s *Store) loadPendingSyncTargets(ctx context.Context, hostID int64) ([]syncTarget, error) {
	var payload []byte
	err := s.db.Pool().QueryRow(ctx, `
		SELECT pending_payload
		FROM santa_sync_state
		WHERE host_id = $1
	`, hostID).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var targets []syncTarget
	if err := json.Unmarshal(payload, &targets); err != nil {
		return nil, err
	}
	return targets, nil
}

func (s *Store) promotePendingSync(
	ctx context.Context,
	hostID int64,
	clientRulesHash string,
	rulesReceived int,
	rulesProcessed int,
) error {
	var pendingCount int
	var pendingFullSync bool
	err := s.db.Pool().QueryRow(ctx, `
		SELECT pending_payload_rule_count, pending_full_sync
		FROM santa_sync_state
		WHERE host_id = $1
	`, hostID).Scan(&pendingCount, &pendingFullSync)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if rulesReceived != pendingCount || rulesProcessed != pendingCount {
		_, err = s.db.Pool().Exec(ctx, `
			UPDATE santa_sync_state
			SET client_rules_hash = $2, updated_at = now()
			WHERE host_id = $1
		`, hostID, clientRulesHash)
		return err
	}
	_, err = s.db.Pool().Exec(ctx, `
		UPDATE santa_sync_state
		SET
			client_rules_hash = $2,
			applied_targets = desired_targets,
			pending_payload = '[]'::jsonb,
			pending_payload_rule_count = 0,
			pending_full_sync = false,
			last_rule_sync_success_at = now(),
			last_clean_sync_at = CASE WHEN $3 THEN now() ELSE last_clean_sync_at END,
			updated_at = now()
		WHERE host_id = $1
	`, hostID, clientRulesHash, pendingFullSync)
	return err
}

func protoRulesFromSyncTargets(targets []syncTarget) []*syncv1.Rule {
	rules := make([]*syncv1.Rule, 0, len(targets))
	for _, target := range targets {
		rules = append(rules, &syncv1.Rule{
			Identifier: target.Identifier,
			RuleType:   protoRuleType(target.RuleType),
			Policy:     protoPolicy(target.Policy),
			CelExpr:    target.CELExpression,
			CustomMsg:  target.CustomMessage,
			CustomUrl:  target.CustomURL,
		})
	}
	return rules
}

func applyConfigurationToPreflightResponse(resp *syncv1.PreflightResponse, config *Configuration) {
	resp.ClientMode = protoClientMode(config.ClientMode)
	resp.EnableBundles = config.EnableBundles
	resp.EnableTransitiveRules = config.EnableTransitiveRules
	resp.EnableAllEventUpload = config.EnableAllEventUpload
	if config.FullSyncIntervalSeconds != nil {
		resp.FullSyncIntervalSeconds = uint32(*config.FullSyncIntervalSeconds)
	}
	if config.BatchSize != nil {
		resp.BatchSize = uint32(*config.BatchSize)
	}
	resp.AllowedPathRegex = config.AllowedPathRegex
	resp.BlockedPathRegex = config.BlockedPathRegex
	resp.EventDetailUrl = config.EventDetailURL
	resp.EventDetailText = config.EventDetailText
	resp.RemovableMediaPolicy = protoRemovableMediaPolicy(config.RemovableMediaAction, config.RemovableMediaRemountFlags)
	resp.EncryptedRemovableMediaPolicy = protoRemovableMediaPolicy(
		config.EncryptedRemovableMediaAction,
		config.EncryptedRemovableMediaRemountFlags,
	)
}

func protoRemovableMediaPolicy(action *RemovableMediaAction, flags []string) *syncv1.RemovableMediaPolicy {
	if action == nil {
		return nil
	}
	switch *action {
	case RemovableMediaActionAllow:
		return &syncv1.RemovableMediaPolicy{Action: &syncv1.RemovableMediaPolicy_Allow{Allow: true}}
	case RemovableMediaActionBlock:
		return &syncv1.RemovableMediaPolicy{Action: &syncv1.RemovableMediaPolicy_Block{Block: true}}
	case RemovableMediaActionRemount:
		return &syncv1.RemovableMediaPolicy{
			Action: &syncv1.RemovableMediaPolicy_Remount{
				Remount: &syncv1.RemountPolicy{Flags: flags},
			},
		}
	default:
		return nil
	}
}

func clientModeFromProto(mode syncv1.ClientMode) ClientMode {
	switch mode {
	case syncv1.ClientMode_MONITOR:
		return ClientModeMonitor
	case syncv1.ClientMode_LOCKDOWN:
		return ClientModeLockdown
	case syncv1.ClientMode_STANDALONE:
		return ClientModeStandalone
	default:
		return ClientModeUnknown
	}
}

func protoClientMode(mode ClientMode) syncv1.ClientMode {
	switch mode {
	case ClientModeMonitor:
		return syncv1.ClientMode_MONITOR
	case ClientModeLockdown:
		return syncv1.ClientMode_LOCKDOWN
	case ClientModeStandalone:
		return syncv1.ClientMode_STANDALONE
	default:
		return syncv1.ClientMode_UNKNOWN_CLIENT_MODE
	}
}

func protoRuleType(ruleType string) syncv1.RuleType {
	switch RuleType(ruleType) {
	case RuleTypeBinary:
		return syncv1.RuleType_BINARY
	case RuleTypeCertificate:
		return syncv1.RuleType_CERTIFICATE
	case RuleTypeTeamID:
		return syncv1.RuleType_TEAMID
	case RuleTypeSigningID:
		return syncv1.RuleType_SIGNINGID
	case RuleTypeCDHash:
		return syncv1.RuleType_CDHASH
	default:
		return syncv1.RuleType_RULETYPE_UNKNOWN
	}
}

func protoPolicy(policy string) syncv1.Policy {
	switch Policy(policy) {
	case PolicyAllowlist:
		return syncv1.Policy_ALLOWLIST
	case PolicyAllowlistCompiler:
		return syncv1.Policy_ALLOWLIST_COMPILER
	case PolicyBlocklist:
		return syncv1.Policy_BLOCKLIST
	case PolicySilentBlocklist:
		return syncv1.Policy_SILENT_BLOCKLIST
	case PolicyCEL:
		return syncv1.Policy_CEL
	default:
		return syncv1.Policy_POLICY_UNKNOWN
	}
}
