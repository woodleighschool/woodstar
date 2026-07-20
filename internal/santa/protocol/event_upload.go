package protocol

import (
	"fmt"
	"math"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa"
	santaevents "github.com/woodleighschool/woodstar/internal/santa/events"
)

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
		BundleHashMillis:        event.GetFileBundleHashMillis(),
		BundleBinaryCount:       event.GetFileBundleBinaryCount(),
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
	case syncv1.Decision_DECISION_UNKNOWN:
		return santaevents.ExecutionDecisionUnknown
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
	case syncv1.FileAccessDecision_FILE_ACCESS_DECISION_UNKNOWN:
		return santaevents.FileAccessDecisionUnknown
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
	case syncv1.SigningStatus_SIGNING_STATUS_UNSPECIFIED:
		return santaevents.SigningStatusUnspecified
	default:
		return santaevents.SigningStatusUnspecified
	}
}
