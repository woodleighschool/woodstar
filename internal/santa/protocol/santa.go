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
	"mime"
	"net/http"
	"strings"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/woodleighschool/woodstar/internal/agentauth"
	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/httpx"
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

type machineIDProtoMessage interface {
	proto.Message
	GetMachineId() string
}

// SyncService handles decoded Santa sync requests.
type SyncService interface {
	Preflight(context.Context, string, santa.PreflightRequest) (santa.PreflightResponse, error)
	EventUpload(context.Context, string, santa.EventUploadRequest) (santa.EventUploadResponse, error)
	RuleDownload(context.Context, string, santa.RuleDownloadRequest) (santa.RuleDownloadResponse, error)
	Postflight(context.Context, string, santa.PostflightRequest) (santa.PostflightResponse, error)
}

type handler struct {
	secretVerifier agentauth.SecretVerifier
	service        SyncService
	logger         *slog.Logger
}

// Server owns Santa sync protocol routes.
type Server struct {
	secretVerifier agentauth.SecretVerifier
	service        SyncService
	logger         *slog.Logger
}

// NewServer returns a Santa sync protocol server.
func NewServer(secretVerifier agentauth.SecretVerifier, service SyncService, logger *slog.Logger) *Server {
	return &Server{secretVerifier: secretVerifier, service: service, logger: logger}
}

// RegisterRoutes mounts Santa sync v1 endpoints on r.
func (s *Server) RegisterRoutes(r chi.Router) {
	h := handler{
		secretVerifier: s.secretVerifier,
		service:        s.service,
		logger:         s.logger,
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

func handleSyncRequest[ProtoReq machineIDProtoMessage, DomainReq any, DomainResp any, ProtoResp proto.Message](
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
	machineID := chi.URLParam(r, "machine_id")
	if err := validateRequestMachineID(machineID, req); err != nil {
		h.writeError(w, r, err)
		return
	}
	domainReq, err := fromProto(req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	resp, err := handle(r.Context(), machineID, domainReq)
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
	ruleCounts, err := ruleCountsFromProto(req)
	if err != nil {
		return santa.PreflightRequest{}, err
	}
	var sipStatus *int16
	if req.GetSipStatus() != 0 {
		value := int16(req.GetSipStatus())
		sipStatus = &value
	}
	return santa.PreflightRequest{
		SerialNumber:      req.GetSerialNumber(),
		Version:           req.GetSantaVersion(),
		RulesHash:         req.GetRulesHash(),
		ClientMode:        clientModeFromProto(req.GetClientMode()),
		RequestCleanSync:  req.GetRequestCleanSync(),
		RuleCounts:        ruleCounts,
		PrimaryUser:       req.GetPrimaryUser(),
		PrimaryUserGroups: req.GetPrimaryUserGroups(),
		SIPStatus:         sipStatus,
	}, nil
}

func preflightResponseToProto(resp santa.PreflightResponse) (*syncv1.PreflightResponse, error) {
	syncType, err := protoSyncType(resp.SyncType)
	if err != nil {
		return nil, err
	}
	out := &syncv1.PreflightResponse{SyncType: &syncType}
	if resp.Configuration != nil {
		if err := applyConfigurationToPreflightResponse(out, resp.Configuration); err != nil {
			return nil, err
		}
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
	standaloneEvents := make([]santaevents.StandaloneRuleCreationEventInput, 0, len(req.GetAuditEvents()))
	for _, event := range req.GetAuditEvents() {
		converted, err := standaloneRuleCreationEventFromProto(event)
		if err != nil {
			return santa.EventUploadRequest{}, err
		}
		standaloneEvents = append(standaloneEvents, converted)
	}
	return santa.EventUploadRequest{
		Events:                       events,
		FileAccessEvents:             fileAccessEvents,
		StandaloneRuleCreationEvents: standaloneEvents,
	}, nil
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
	rulesReceived, err := int32FromUint32(req.GetRulesReceived(), "rules_received")
	if err != nil {
		return santa.PostflightRequest{}, err
	}
	rulesProcessed, err := int32FromUint32(req.GetRulesProcessed(), "rules_processed")
	if err != nil {
		return santa.PostflightRequest{}, err
	}
	syncType, err := syncTypeFromProto(req.GetSyncType())
	if err != nil {
		return santa.PostflightRequest{}, err
	}
	return santa.PostflightRequest{
		RulesReceived:  rulesReceived,
		RulesProcessed: rulesProcessed,
		SyncType:       syncType,
		RulesHash:      req.GetRulesHash(),
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
		occurredAt, err = requiredUnixSecondsToTime(event.GetExecutionTime(), "execution_time")
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
		StaticRule:              event.GetStaticRule(),
		BundleID:                event.GetFileBundleId(),
		BundlePath:              event.GetFileBundlePath(),
		BundleExecutableRelPath: event.GetFileBundleExecutableRelPath(),
		BundleName:              event.GetFileBundleName(),
		BundleVersion:           event.GetFileBundleVersion(),
		BundleVersionString:     event.GetFileBundleVersionString(),
		BundleHash:              event.GetFileBundleHash(),
		BundleHashMillis:        int32(event.GetFileBundleHashMillis()),
		BundleBinaryCount:       int32(event.GetFileBundleBinaryCount()),
		PID:                     event.GetPid(),
		PPID:                    event.GetPpid(),
		ParentName:              event.GetParentName(),
		SigningID:               event.GetSigningId(),
		TeamID:                  event.GetTeamId(),
		CDHash:                  event.GetCdhash(),
		CodesigningFlags:        event.GetCsFlags(),
		SigningStatus:           signingStatusFromProto(event.GetSigningStatus()),
		SecureSigningTime:       optionalUnixSecondsToTime(event.GetSecureSigningTime()),
		SigningTime:             optionalUnixSecondsToTime(event.GetSigningTime()),
		Entitlements:            entitlements,
		SigningChain:            signingChainFromProto(event.GetSigningChain()),
	}, nil
}

func standaloneRuleCreationEventFromProto(
	event *syncv1.AuditEvent,
) (santaevents.StandaloneRuleCreationEventInput, error) {
	if event == nil {
		return santaevents.StandaloneRuleCreationEventInput{}, fmt.Errorf(
			"%w: audit event is required",
			dbutil.ErrInvalidInput,
		)
	}
	creation := event.GetStandaloneModeRuleCreation()
	if creation == nil {
		return santaevents.StandaloneRuleCreationEventInput{}, fmt.Errorf(
			"%w: unsupported audit event",
			dbutil.ErrInvalidInput,
		)
	}
	occurredAt, err := requiredUnixSecondsToTime(float64(creation.GetTimestamp()), "audit_event.timestamp")
	if err != nil {
		return santaevents.StandaloneRuleCreationEventInput{}, err
	}
	return santaevents.StandaloneRuleCreationEventInput{
		Identifier: creation.GetIdentifier(),
		Decision:   decisionFromProto(creation.GetDecision()),
		OccurredAt: occurredAt,
	}, nil
}

func ruleCountsFromProto(req *syncv1.PreflightRequest) (syncstate.RuleCounts, error) {
	values := []struct {
		name  string
		value uint32
	}{
		{name: "binary_rule_count", value: req.GetBinaryRuleCount()},
		{name: "certificate_rule_count", value: req.GetCertificateRuleCount()},
		{name: "teamid_rule_count", value: req.GetTeamidRuleCount()},
		{name: "signingid_rule_count", value: req.GetSigningidRuleCount()},
		{name: "cdhash_rule_count", value: req.GetCdhashRuleCount()},
		{name: "compiler_rule_count", value: req.GetCompilerRuleCount()},
		{name: "transitive_rule_count", value: req.GetTransitiveRuleCount()},
	}
	counts := make([]int32, len(values))
	for i, value := range values {
		converted, err := int32FromUint32(value.value, value.name)
		if err != nil {
			return syncstate.RuleCounts{}, err
		}
		counts[i] = converted
	}
	return syncstate.RuleCounts{
		Binary:      counts[0],
		Certificate: counts[1],
		TeamID:      counts[2],
		SigningID:   counts[3],
		CDHash:      counts[4],
		Compiler:    counts[5],
		Transitive:  counts[6],
	}, nil
}

func int32FromUint32(value uint32, field string) (int32, error) {
	if value > math.MaxInt32 {
		return 0, fmt.Errorf("%w: %s exceeds supported range", dbutil.ErrInvalidInput, field)
	}
	return int32(value), nil
}

func syncTypeFromProto(value syncv1.SyncType) (syncstate.SyncType, error) {
	switch value {
	case syncv1.SyncType_NORMAL:
		return syncstate.SyncTypeNormal, nil
	case syncv1.SyncType_CLEAN:
		return syncstate.SyncTypeClean, nil
	case syncv1.SyncType_CLEAN_ALL:
		return syncstate.SyncTypeCleanAll, nil
	default:
		return "", fmt.Errorf("%w: unsupported sync_type %q", dbutil.ErrInvalidInput, value)
	}
}

func entitlementJSON(event *syncv1.Event) ([]byte, error) {
	entitlements := event.GetEntitlementInfo()
	if entitlements == nil {
		return nil, nil
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
	occurredAt, err := requiredUnixSecondsToTime(event.GetAccessTime(), "access_time")
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

func requiredUnixSecondsToTime(seconds float64, field string) (time.Time, error) {
	if seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return time.Time{}, fmt.Errorf("%w: %s is required", dbutil.ErrInvalidInput, field)
	}
	return unixSecondsToTime(seconds), nil
}

func optionalUnixSecondsToTime(seconds uint32) time.Time {
	if seconds == 0 {
		return time.Time{}
	}
	return unixSecondsToTime(float64(seconds))
}

func unixSecondsToTime(seconds float64) time.Time {
	whole, fraction := math.Modf(seconds)
	return time.Unix(int64(whole), int64(fraction*1e9)).UTC()
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
	token, ok := httpx.BearerToken(r.Header.Get("Authorization"))
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
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != protobufContentType {
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

func validateRequestMachineID(pathMachineID string, req machineIDProtoMessage) error {
	if req.GetMachineId() != pathMachineID {
		return fmt.Errorf(
			"%w: body machine_id %q does not match path machine_id %q",
			dbutil.ErrInvalidInput,
			req.GetMachineId(),
			pathMachineID,
		)
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

func applyConfigurationToPreflightResponse(resp *syncv1.PreflightResponse, config *configurations.Configuration) error {
	clientMode, err := protoClientMode(config.ClientMode)
	if err != nil {
		return err
	}
	resp.ClientMode = clientMode
	resp.EnableBundles = &config.EnableBundles
	resp.EnableTransitiveRules = &config.EnableTransitiveRules
	resp.EnableAllEventUpload = &config.EnableAllEventUpload
	resp.DisableUnknownEventUpload = &config.DisableUnknownEventUpload
	fileAccessAction, err := protoFileAccessAction(config.OverrideFileAccessAction)
	if err != nil {
		return err
	}
	resp.OverrideFileAccessAction = &fileAccessAction
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
	return nil
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

func protoFileAccessAction(action configurations.FileAccessAction) (syncv1.FileAccessAction, error) {
	switch action {
	case configurations.FileAccessActionNone:
		return syncv1.FileAccessAction_NONE, nil
	case configurations.FileAccessActionAuditOnly:
		return syncv1.FileAccessAction_AUDIT_ONLY, nil
	case configurations.FileAccessActionDisable:
		return syncv1.FileAccessAction_DISABLE, nil
	default:
		return syncv1.FileAccessAction_FILE_ACCESS_ACTION_UNSPECIFIED, fmt.Errorf(
			"%w: unsupported override_file_access_action %q",
			dbutil.ErrInvalidInput,
			action,
		)
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

func protoClientMode(mode configurations.ClientMode) (syncv1.ClientMode, error) {
	switch mode {
	case configurations.ClientModeMonitor:
		return syncv1.ClientMode_MONITOR, nil
	case configurations.ClientModeLockdown:
		return syncv1.ClientMode_LOCKDOWN, nil
	case configurations.ClientModeStandalone:
		return syncv1.ClientMode_STANDALONE, nil
	default:
		return syncv1.ClientMode_UNKNOWN_CLIENT_MODE, fmt.Errorf(
			"%w: unsupported client_mode %q",
			dbutil.ErrInvalidInput,
			mode,
		)
	}
}

func protoSyncType(syncType syncstate.SyncType) (syncv1.SyncType, error) {
	switch syncType {
	case syncstate.SyncTypeNormal:
		return syncv1.SyncType_NORMAL, nil
	case syncstate.SyncTypeClean:
		return syncv1.SyncType_CLEAN, nil
	case syncstate.SyncTypeCleanAll:
		return syncv1.SyncType_CLEAN_ALL, nil
	default:
		return syncv1.SyncType_SYNC_TYPE_UNSPECIFIED, fmt.Errorf(
			"%w: unsupported sync_type %q",
			dbutil.ErrInvalidInput,
			syncType,
		)
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
	case santarules.PolicySilentGUIBlocklist:
		return syncv1.Policy_SILENT_GUI_BLOCKLIST
	case santarules.PolicySilentTTYBlocklist:
		return syncv1.Policy_SILENT_TTY_BLOCKLIST
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
	case syncv1.Decision_BLOCK_BINARY_MISMATCH:
		return santaevents.ExecutionDecisionBlockBinaryMismatch
	case syncv1.Decision_ALLOW_PLATFORM:
		return santaevents.ExecutionDecisionAllowPlatform
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
	case errors.Is(err, dbutil.ErrNotFound):
		return http.StatusNotFound
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
