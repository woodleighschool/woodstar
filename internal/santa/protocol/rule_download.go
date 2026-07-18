package protocol

import (
	"fmt"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"

	"github.com/woodleighschool/woodstar/internal/dbutil"
	"github.com/woodleighschool/woodstar/internal/santa"
	santarules "github.com/woodleighschool/woodstar/internal/santa/rules"
	"github.com/woodleighschool/woodstar/internal/santa/syncstate"
)

func ruleDownloadRequestFromProto(req *syncv1.RuleDownloadRequest) (santa.RuleDownloadRequest, error) {
	return santa.RuleDownloadRequest{Cursor: req.GetCursor()}, nil
}

func ruleDownloadResponseToProto(resp santa.RuleDownloadResponse) (*syncv1.RuleDownloadResponse, error) {
	rules, err := protoRulesFromPayloadRules(resp.Rules)
	if err != nil {
		return nil, err
	}
	return &syncv1.RuleDownloadResponse{
		Rules:  rules,
		Cursor: resp.Cursor,
	}, nil
}

func protoRulesFromPayloadRules(payload []syncstate.PayloadRule) ([]*syncv1.Rule, error) {
	rules := make([]*syncv1.Rule, 0, len(payload))
	for _, rule := range payload {
		ruleType, err := protoRuleType(rule.RuleType)
		if err != nil {
			return nil, err
		}
		rules = append(rules, &syncv1.Rule{
			Identifier:          rule.Identifier,
			RuleType:            ruleType,
			Policy:              protoPolicy(rule),
			CelExpr:             rule.CELExpression,
			CustomMsg:           rule.CustomMessage,
			CustomUrl:           rule.CustomURL,
			NotificationAppName: rule.AppName,
		})
	}
	return rules, nil
}

func protoRuleType(ruleType string) (syncv1.RuleType, error) {
	switch santarules.RuleType(ruleType) {
	case santarules.RuleTypeBinary:
		return syncv1.RuleType_BINARY, nil
	case santarules.RuleTypeCertificate:
		return syncv1.RuleType_CERTIFICATE, nil
	case santarules.RuleTypeTeamID:
		return syncv1.RuleType_TEAMID, nil
	case santarules.RuleTypeSigningID:
		return syncv1.RuleType_SIGNINGID, nil
	case santarules.RuleTypeCDHash:
		return syncv1.RuleType_CDHASH, nil
	case santarules.RuleTypeBundle:
		return syncv1.RuleType_RULETYPE_UNKNOWN, fmt.Errorf(
			"%w: bundle rules must be expanded before download",
			dbutil.ErrInvalidInput,
		)
	default:
		return syncv1.RuleType_RULETYPE_UNKNOWN, fmt.Errorf(
			"%w: unsupported rule type %q",
			dbutil.ErrInvalidInput,
			ruleType,
		)
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
