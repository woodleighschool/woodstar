// Package protocol exposes Santa sync endpoints.
package protocol

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa"
	"github.com/woodleighschool/woodstar/internal/santa/configurations"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

const (
	protobufContentType = "application/x-protobuf"
	maxRequestBodyBytes = 16 << 20
)

var (
	errUnauthorized      = errors.New("unauthorized santa sync request")
	errUnsupportedMedia  = errors.New("unsupported santa sync media")
	errInvalidSyncBody   = errors.New("invalid santa sync request body")
	errRequestBodyTooBig = errors.New("santa sync request body too large")
)

// AgentSecretVerifier verifies shared agent secrets parsed from bearer authorization.
type AgentSecretVerifier interface {
	Verify(context.Context, agentauth.Agent, string) (bool, error)
}

// SyncService handles decoded Santa sync requests.
type SyncService interface {
	Preflight(context.Context, string, santa.PreflightRequest) (santa.PreflightResponse, error)
	EventUpload(context.Context, string, santa.EventUploadRequest) (santa.EventUploadResponse, error)
	RuleDownload(context.Context, string, santa.RuleDownloadRequest) (santa.RuleDownloadResponse, error)
	Postflight(context.Context, string, santa.PostflightRequest) (santa.PostflightResponse, error)
}

type handler struct {
	secretVerifier AgentSecretVerifier
	service        SyncService
	logger         *slog.Logger
}

// RegisterSantaRoutes mounts Santa sync v1 endpoints on r.
func RegisterSantaRoutes(r chi.Router, secretVerifier AgentSecretVerifier, service SyncService, logger *slog.Logger) {
	h := handler{
		secretVerifier: secretVerifier,
		service:        service,
		logger:         logger,
	}
	r.Post("/santa/sync/preflight/{machine_id}", h.preflight)
	r.Post("/santa/sync/eventupload/{machine_id}", h.eventUpload)
	r.Post("/santa/sync/ruledownload/{machine_id}", h.ruleDownload)
	r.Post("/santa/sync/postflight/{machine_id}", h.postflight)
}

func (h handler) preflight(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(
		h,
		w,
		r,
		&syncv1.PreflightRequest{},
		preflightRequestFromProto,
		h.service.Preflight,
		preflightResponseToProto,
	)
}

func (h handler) eventUpload(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(
		h,
		w,
		r,
		&syncv1.EventUploadRequest{},
		eventUploadRequestFromProto,
		h.service.EventUpload,
		eventUploadResponseToProto,
	)
}

func (h handler) ruleDownload(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(
		h,
		w,
		r,
		&syncv1.RuleDownloadRequest{},
		ruleDownloadRequestFromProto,
		h.service.RuleDownload,
		ruleDownloadResponseToProto,
	)
}

func (h handler) postflight(w http.ResponseWriter, r *http.Request) {
	handleSyncRequest(
		h,
		w,
		r,
		&syncv1.PostflightRequest{},
		postflightRequestFromProto,
		h.service.Postflight,
		postflightResponseToProto,
	)
}

func handleSyncRequest[ProtoReq proto.Message, DomainReq any, DomainResp any, ProtoResp proto.Message](
	h handler,
	w http.ResponseWriter,
	r *http.Request,
	req ProtoReq,
	fromProto func(ProtoReq) (DomainReq, error),
	handle func(context.Context, string, DomainReq) (DomainResp, error),
	toProto func(DomainResp) (ProtoResp, error),
) {
	if err := h.authorize(r); err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := validateTransportHeaders(r); err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := decodeRequest(r, req); err != nil {
		h.writeError(w, r, err)
		return
	}
	domainReq, err := fromProto(req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	resp, err := handle(r.Context(), chi.URLParam(r, "machine_id"), domainReq)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	protoResp, err := toProto(resp)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := writeProtoResponse(w, protoResp); err != nil {
		h.log(r, http.StatusInternalServerError, err)
		writeStatusOnly(w, http.StatusInternalServerError)
	}
}

func preflightRequestFromProto(req *syncv1.PreflightRequest) (santa.PreflightRequest, error) {
	var sipStatus *int16
	if req.GetSipStatus() != 0 {
		value := int16(req.GetSipStatus())
		sipStatus = &value
	}
	return santa.PreflightRequest{
		SerialNumber:     req.GetSerialNumber(),
		Version:          req.GetSantaVersion(),
		ClientMode:       clientModeFromProto(req.GetClientMode()),
		RequestCleanSync: req.GetRequestCleanSync(),
		RulesHash:        req.GetRulesHash(),
		RuleCounts: syncstate.RuleCounts{
			Binary:      int(req.GetBinaryRuleCount()),
			Certificate: int(req.GetCertificateRuleCount()),
			TeamID:      int(req.GetTeamidRuleCount()),
			SigningID:   int(req.GetSigningidRuleCount()),
			CDHash:      int(req.GetCdhashRuleCount()),
		},
		PrimaryUser:       req.GetPrimaryUser(),
		PrimaryUserGroups: req.GetPrimaryUserGroups(),
		SIPStatus:         sipStatus,
		OSBuild:           req.GetOsBuild(),
		ModelIdentifier:   req.GetModelIdentifier(),
	}, nil
}

func preflightResponseToProto(resp santa.PreflightResponse) (*syncv1.PreflightResponse, error) {
	syncType := protoSyncType(resp.SyncType)
	out := &syncv1.PreflightResponse{SyncType: &syncType}
	if resp.Configuration != nil {
		applyConfigurationToPreflightResponse(out, resp.Configuration)
	}
	return out, nil
}

func eventUploadRequestFromProto(req *syncv1.EventUploadRequest) (santa.EventUploadRequest, error) {
	events := make([]santaevents.ExecutionEventInput, 0, len(req.GetEvents()))
	for _, event := range req.GetEvents() {
		if event == nil {
			continue
		}
		converted, err := executionEventFromProto(event)
		if err != nil {
			return santa.EventUploadRequest{}, err
		}
		events = append(events, converted)
	}
	fileAccessEvents := make([]santaevents.FileAccessEventInput, 0, len(req.GetFileAccessEvents()))
	for _, event := range req.GetFileAccessEvents() {
		if event == nil {
			continue
		}
		converted, err := fileAccessEventFromProto(event)
		if err != nil {
			return santa.EventUploadRequest{}, err
		}
		fileAccessEvents = append(fileAccessEvents, converted)
	}
	return santa.EventUploadRequest{Events: events, FileAccessEvents: fileAccessEvents}, nil
}

func eventUploadResponseToProto(resp santa.EventUploadResponse) (*syncv1.EventUploadResponse, error) {
	return &syncv1.EventUploadResponse{EventUploadBundleBinaries: resp.BundleBinaryRequests}, nil
}

func ruleDownloadRequestFromProto(req *syncv1.RuleDownloadRequest) (santa.RuleDownloadRequest, error) {
	return santa.RuleDownloadRequest{Cursor: req.GetCursor()}, nil
}

func ruleDownloadResponseToProto(resp santa.RuleDownloadResponse) (*syncv1.RuleDownloadResponse, error) {
	return &syncv1.RuleDownloadResponse{
		Rules:  protoRulesFromPayloadRules(resp.Rules),
		Cursor: resp.Cursor,
	}, nil
}

func postflightRequestFromProto(req *syncv1.PostflightRequest) (santa.PostflightRequest, error) {
	return santa.PostflightRequest{
		RulesHash:      req.GetRulesHash(),
		RulesReceived:  int(req.GetRulesReceived()),
		RulesProcessed: int(req.GetRulesProcessed()),
	}, nil
}

func postflightResponseToProto(santa.PostflightResponse) (*syncv1.PostflightResponse, error) {
	return &syncv1.PostflightResponse{}, nil
}

func executionEventFromProto(event *syncv1.Event) (santaevents.ExecutionEventInput, error) {
	entitlements, err := entitlementJSON(event)
	if err != nil {
		return santaevents.ExecutionEventInput{}, err
	}
	decision := decisionFromProto(event.GetDecision())
	var occurredAt time.Time
	if decision != santaevents.ExecutionDecisionBundleBinary {
		occurredAt, err = eventTime(event.GetExecutionTime(), "execution_time")
		if err != nil {
			return santaevents.ExecutionEventInput{}, err
		}
	}
	return santaevents.ExecutionEventInput{
		FileSHA256:              event.GetFileSha256(),
		FilePath:                event.GetFilePath(),
		FileName:                event.GetFileName(),
		ExecutingUser:           event.GetExecutingUser(),
		OccurredAt:              occurredAt,
		LoggedInUsers:           event.GetLoggedInUsers(),
		CurrentSessions:         event.GetCurrentSessions(),
		Decision:                decision,
		BundleID:                event.GetFileBundleId(),
		BundlePath:              event.GetFileBundlePath(),
		BundleExecutableRelPath: event.GetFileBundleExecutableRelPath(),
		BundleName:              event.GetFileBundleName(),
		BundleVersion:           event.GetFileBundleVersion(),
		BundleVersionString:     event.GetFileBundleVersionString(),
		BundleHash:              event.GetFileBundleHash(),
		BundleHashMillis:        int(event.GetFileBundleHashMillis()),
		BundleBinaryCount:       int(event.GetFileBundleBinaryCount()),
		PID:                     event.GetPid(),
		PPID:                    event.GetPpid(),
		ParentName:              event.GetParentName(),
		SigningID:               event.GetSigningId(),
		TeamID:                  event.GetTeamId(),
		CDHash:                  event.GetCdhash(),
		CodesigningFlags:        event.GetCsFlags(),
		SigningStatus:           signingStatusFromProto(event.GetSigningStatus()),
		SecureSigningTime:       eventTimestamp(event.GetSecureSigningTime()),
		SigningTime:             eventTimestamp(event.GetSigningTime()),
		Entitlements:            entitlements,
		SigningChain:            signingChainFromProto(event.GetSigningChain()),
	}, nil
}

func entitlementJSON(event *syncv1.Event) ([]byte, error) {
	entitlements := event.GetEntitlementInfo()
	if entitlements == nil {
		return []byte{}, nil
	}
	return protojson.Marshal(entitlements)
}

func signingChainFromProto(chain []*syncv1.Certificate) []santaevents.CertificateInput {
	out := make([]santaevents.CertificateInput, 0, len(chain))
	for _, cert := range chain {
		if cert == nil {
			continue
		}
		out = append(out, santaevents.CertificateInput{
			SHA256:     cert.GetSha256(),
			CommonName: cert.GetCn(),
			Org:        cert.GetOrg(),
			OU:         cert.GetOu(),
			ValidFrom:  cert.GetValidFrom(),
			ValidUntil: cert.GetValidUntil(),
		})
	}
	return out
}

func fileAccessEventFromProto(event *syncv1.FileAccessEvent) (santaevents.FileAccessEventInput, error) {
	occurredAt, err := eventTime(event.GetAccessTime(), "access_time")
	if err != nil {
		return santaevents.FileAccessEventInput{}, err
	}
	return santaevents.FileAccessEventInput{
		RuleVersion:  event.GetRuleVersion(),
		RuleName:     event.GetRuleName(),
		Target:       event.GetTarget(),
		Decision:     fileAccessDecisionFromProto(event.GetDecision()),
		OccurredAt:   occurredAt,
		ProcessChain: processChainFromProto(event.GetProcessChain()),
	}, nil
}

func eventTime(seconds float64, field string) (time.Time, error) {
	if seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return time.Time{}, fmt.Errorf("%w: %s is required", dbutil.ErrInvalidInput, field)
	}
	whole, fraction := math.Modf(seconds)
	return time.Unix(int64(whole), int64(fraction*1e9)).UTC(), nil
}

func eventTimestamp(seconds uint32) time.Time {
	if seconds == 0 {
		return time.Time{}
	}
	return time.Unix(int64(seconds), 0).UTC()
}

func processChainFromProto(processes []*syncv1.Process) []santaevents.ProcessInput {
	out := make([]santaevents.ProcessInput, 0, len(processes))
	for _, process := range processes {
		if process == nil {
			continue
		}
		out = append(out, santaevents.ProcessInput{
			PID:          process.GetPid(),
			FilePath:     process.GetFilePath(),
			FileSHA256:   process.GetFileSha256(),
			SigningID:    process.GetSigningId(),
			TeamID:       process.GetTeamId(),
			CDHash:       process.GetCdhash(),
			SigningChain: signingChainFromProto(process.GetSigningChain()),
		})
	}
	return out
}

func (h handler) authorize(r *http.Request) error {
	token, ok := agentauth.BearerToken(r.Header.Get("Authorization"))
	if !ok {
		return errUnauthorized
	}

	ok, err := h.secretVerifier.Verify(r.Context(), agentauth.AgentSanta, token)
	if err != nil {
		return err
	}
	if !ok {
		return errUnauthorized
	}
	return nil
}

func validateTransportHeaders(r *http.Request) error {
	if r.Header.Get("Content-Type") != protobufContentType {
		return errUnsupportedMedia
	}
	if !strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		return errUnsupportedMedia
	}
	return nil
}

func decodeRequest(r *http.Request, msg proto.Message) error {
	zr, err := gzip.NewReader(r.Body)
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidSyncBody, err)
	}
	defer zr.Close()

	payload, err := io.ReadAll(io.LimitReader(zr, maxRequestBodyBytes+1))
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidSyncBody, err)
	}
	if len(payload) > maxRequestBodyBytes {
		return errRequestBodyTooBig
	}
	if err := proto.Unmarshal(payload, msg); err != nil {
		return fmt.Errorf("%w: %w", errInvalidSyncBody, err)
	}
	return nil
}

func writeProtoResponse(w http.ResponseWriter, msg proto.Message) error {
	payload, err := marshalCompressedProto(msg)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", protobufContentType)
	w.Header().Set("Content-Encoding", "gzip")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(payload)
	return err
}

func marshalCompressedProto(msg proto.Message) ([]byte, error) {
	payload, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(payload); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func applyConfigurationToPreflightResponse(resp *syncv1.PreflightResponse, config *configurations.Configuration) {
	resp.ClientMode = protoClientMode(config.ClientMode)
	resp.EnableBundles = &config.EnableBundles
	resp.EnableTransitiveRules = &config.EnableTransitiveRules
	resp.EnableAllEventUpload = &config.EnableAllEventUpload
	resp.FullSyncIntervalSeconds = uint32(config.FullSyncIntervalSeconds)
	resp.BatchSize = uint32(config.BatchSize)
	if config.AllowedPathRegex != "" {
		resp.AllowedPathRegex = &config.AllowedPathRegex
	}
	if config.BlockedPathRegex != "" {
		resp.BlockedPathRegex = &config.BlockedPathRegex
	}
	if config.EventDetailURL != "" {
		resp.EventDetailUrl = &config.EventDetailURL
	}
	if config.EventDetailText != "" {
		resp.EventDetailText = &config.EventDetailText
	}
	resp.RemovableMediaPolicy = protoRemovableMediaPolicy(config.RemovableMediaPolicy)
	resp.EncryptedRemovableMediaPolicy = protoRemovableMediaPolicy(config.EncryptedRemovableMediaPolicy)
}

func protoRemovableMediaPolicy(policy configurations.RemovableMediaPolicy) *syncv1.RemovableMediaPolicy {
	switch policy.Action {
	case configurations.RemovableMediaActionAllow:
		return &syncv1.RemovableMediaPolicy{Action: &syncv1.RemovableMediaPolicy_Allow{Allow: true}}
	case configurations.RemovableMediaActionBlock:
		return &syncv1.RemovableMediaPolicy{Action: &syncv1.RemovableMediaPolicy_Block{Block: true}}
	case configurations.RemovableMediaActionRemount:
		return &syncv1.RemovableMediaPolicy{
			Action: &syncv1.RemovableMediaPolicy_Remount{
				Remount: &syncv1.RemountPolicy{Flags: policy.RemountFlags},
			},
		}
	default:
		return nil
	}
}

func clientModeFromProto(mode syncv1.ClientMode) configurations.ReportedClientMode {
	switch mode {
	case syncv1.ClientMode_MONITOR:
		return configurations.ReportedClientModeMonitor
	case syncv1.ClientMode_LOCKDOWN:
		return configurations.ReportedClientModeLockdown
	case syncv1.ClientMode_STANDALONE:
		return configurations.ReportedClientModeStandalone
	default:
		return configurations.ReportedClientModeUnknown
	}
}

func protoClientMode(mode configurations.ClientMode) syncv1.ClientMode {
	switch mode {
	case configurations.ClientModeMonitor:
		return syncv1.ClientMode_MONITOR
	case configurations.ClientModeLockdown:
		return syncv1.ClientMode_LOCKDOWN
	case configurations.ClientModeStandalone:
		return syncv1.ClientMode_STANDALONE
	default:
		return syncv1.ClientMode_UNKNOWN_CLIENT_MODE
	}
}

func protoSyncType(syncType syncstate.SyncType) syncv1.SyncType {
	switch syncType {
	case syncstate.SyncTypeClean:
		return syncv1.SyncType_CLEAN
	default:
		return syncv1.SyncType_NORMAL
	}
}

func protoRulesFromPayloadRules(payload []syncstate.PayloadRule) []*syncv1.Rule {
	rules := make([]*syncv1.Rule, 0, len(payload))
	for _, rule := range payload {
		rules = append(rules, &syncv1.Rule{
			Identifier:          rule.Identifier,
			RuleType:            protoRuleType(rule.RuleType),
			Policy:              protoPolicy(rule),
			CelExpr:             rule.CELExpression,
			CustomMsg:           rule.CustomMessage,
			CustomUrl:           rule.CustomURL,
			NotificationAppName: rule.AppName,
		})
	}
	return rules
}

func protoRuleType(ruleType string) syncv1.RuleType {
	switch santarules.RuleType(ruleType) {
	case santarules.RuleTypeBinary:
		return syncv1.RuleType_BINARY
	case santarules.RuleTypeCertificate:
		return syncv1.RuleType_CERTIFICATE
	case santarules.RuleTypeTeamID:
		return syncv1.RuleType_TEAMID
	case santarules.RuleTypeSigningID:
		return syncv1.RuleType_SIGNINGID
	case santarules.RuleTypeCDHash:
		return syncv1.RuleType_CDHASH
	default:
		return syncv1.RuleType_RULETYPE_UNKNOWN
	}
}

func protoPolicy(rule syncstate.PayloadRule) syncv1.Policy {
	if rule.Removed {
		return syncv1.Policy_REMOVE
	}
	switch santarules.Policy(rule.Policy) {
	case santarules.PolicyAllowlist:
		return syncv1.Policy_ALLOWLIST
	case santarules.PolicyAllowlistCompiler:
		return syncv1.Policy_ALLOWLIST_COMPILER
	case santarules.PolicyBlocklist:
		return syncv1.Policy_BLOCKLIST
	case santarules.PolicySilentBlocklist:
		return syncv1.Policy_SILENT_BLOCKLIST
	case santarules.PolicyCEL:
		return syncv1.Policy_CEL
	default:
		return syncv1.Policy_POLICY_UNKNOWN
	}
}

func decisionFromProto(decision syncv1.Decision) santaevents.ExecutionDecision {
	switch decision {
	case syncv1.Decision_ALLOW_UNKNOWN:
		return santaevents.ExecutionDecisionAllowUnknown
	case syncv1.Decision_ALLOW_BINARY:
		return santaevents.ExecutionDecisionAllowBinary
	case syncv1.Decision_ALLOW_CERTIFICATE:
		return santaevents.ExecutionDecisionAllowCertificate
	case syncv1.Decision_ALLOW_SCOPE:
		return santaevents.ExecutionDecisionAllowScope
	case syncv1.Decision_ALLOW_TEAMID:
		return santaevents.ExecutionDecisionAllowTeamID
	case syncv1.Decision_ALLOW_SIGNINGID:
		return santaevents.ExecutionDecisionAllowSigningID
	case syncv1.Decision_ALLOW_CDHASH:
		return santaevents.ExecutionDecisionAllowCDHash
	case syncv1.Decision_BLOCK_UNKNOWN:
		return santaevents.ExecutionDecisionBlockUnknown
	case syncv1.Decision_BLOCK_BINARY:
		return santaevents.ExecutionDecisionBlockBinary
	case syncv1.Decision_BLOCK_CERTIFICATE:
		return santaevents.ExecutionDecisionBlockCertificate
	case syncv1.Decision_BLOCK_SCOPE:
		return santaevents.ExecutionDecisionBlockScope
	case syncv1.Decision_BLOCK_TEAMID:
		return santaevents.ExecutionDecisionBlockTeamID
	case syncv1.Decision_BLOCK_SIGNINGID:
		return santaevents.ExecutionDecisionBlockSigningID
	case syncv1.Decision_BLOCK_CDHASH:
		return santaevents.ExecutionDecisionBlockCDHash
	case syncv1.Decision_BUNDLE_BINARY:
		return santaevents.ExecutionDecisionBundleBinary
	default:
		return santaevents.ExecutionDecisionUnknown
	}
}

func fileAccessDecisionFromProto(decision syncv1.FileAccessDecision) santaevents.FileAccessDecision {
	switch decision {
	case syncv1.FileAccessDecision_FILE_ACCESS_DECISION_DENIED:
		return santaevents.FileAccessDecisionDenied
	case syncv1.FileAccessDecision_FILE_ACCESS_DECISION_DENIED_INVALID_SIGNATURE:
		return santaevents.FileAccessDecisionDeniedInvalidSignature
	case syncv1.FileAccessDecision_FILE_ACCESS_DECISION_AUDIT_ONLY:
		return santaevents.FileAccessDecisionAuditOnly
	default:
		return santaevents.FileAccessDecisionUnknown
	}
}

func signingStatusFromProto(status syncv1.SigningStatus) santaevents.SigningStatus {
	switch status {
	case syncv1.SigningStatus_SIGNING_STATUS_UNSIGNED:
		return santaevents.SigningStatusUnsigned
	case syncv1.SigningStatus_SIGNING_STATUS_INVALID:
		return santaevents.SigningStatusInvalid
	case syncv1.SigningStatus_SIGNING_STATUS_ADHOC:
		return santaevents.SigningStatusAdhoc
	case syncv1.SigningStatus_SIGNING_STATUS_DEVELOPMENT:
		return santaevents.SigningStatusDevelopment
	case syncv1.SigningStatus_SIGNING_STATUS_PRODUCTION:
		return santaevents.SigningStatusProduction
	default:
		return santaevents.SigningStatusUnspecified
	}
}

func (h handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	statusCode := statusCodeForError(err)
	h.log(r, statusCode, err)
	writeStatusOnly(w, statusCode)
}

func statusCodeForError(err error) int {
	switch {
	case errors.Is(err, errUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, errUnsupportedMedia):
		return http.StatusUnsupportedMediaType
	case errors.Is(err, errInvalidSyncBody),
		errors.Is(err, errRequestBodyTooBig),
		errors.Is(err, dbutil.ErrInvalidInput):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeStatusOnly(w http.ResponseWriter, statusCode int) {
	w.Header().Del("Content-Type")
	w.Header().Del("Content-Encoding")
	w.WriteHeader(statusCode)
}

func (h handler) log(r *http.Request, statusCode int, err error) {
	if h.logger == nil {
		return
	}
	args := []any{
		"status", statusCode,
		"method", r.Method,
		"path", r.URL.Path,
		"machine_id", chi.URLParam(r, "machine_id"),
		"err", err,
	}
	if statusCode >= http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "santa sync request failed", args...)
		return
	}
	h.logger.WarnContext(r.Context(), "santa sync request rejected", args...)
}
