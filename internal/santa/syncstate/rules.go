package syncstate

import (
	"cmp"
	"slices"
)

type Target struct {
	RuleType      string `json:"rule_type"`
	Identifier    string `json:"identifier"`
	Policy        string `json:"policy"`
	CELExpression string `json:"cel_expression,omitempty"`
	CustomMessage string `json:"custom_message,omitempty"`
	CustomURL     string `json:"custom_url,omitempty"`
	AppName       string `json:"notification_app_name,omitempty"`
	PayloadHash   string `json:"payload_hash"`
}

type PayloadRule struct {
	RuleType      string `json:"rule_type"`
	Identifier    string `json:"identifier"`
	Policy        string `json:"policy,omitempty"`
	CELExpression string `json:"cel_expression,omitempty"`
	CustomMessage string `json:"custom_message,omitempty"`
	CustomURL     string `json:"custom_url,omitempty"`
	AppName       string `json:"notification_app_name,omitempty"`
	PayloadHash   string `json:"payload_hash,omitempty"`
	Removed       bool   `json:"removed,omitempty"`
}

func (target Target) Key() string {
	return target.RuleType + "\x00" + target.Identifier + "\x00" + target.PayloadHash
}

func (target Target) identityKey() string {
	return target.RuleType + "\x00" + target.Identifier
}

func TargetSet(targets []Target) map[string]bool {
	out := make(map[string]bool, len(targets))
	for _, target := range targets {
		out[target.Key()] = true
	}
	return out
}

func normalSyncPayload(desired []Target, applied []Target) []PayloadRule {
	appliedByIdentity := make(map[string]Target, len(applied))
	for _, target := range applied {
		appliedByIdentity[target.identityKey()] = target
	}
	currentIdentities := make(map[string]struct{}, len(desired))
	payload := make([]PayloadRule, 0, len(desired)+len(applied))
	for _, target := range desired {
		currentIdentities[target.identityKey()] = struct{}{}
		current, ok := appliedByIdentity[target.identityKey()]
		if !ok || current.PayloadHash != target.PayloadHash {
			payload = append(payload, payloadRuleFromTarget(target))
		}
	}

	for _, target := range applied {
		if _, ok := currentIdentities[target.identityKey()]; ok {
			continue
		}
		payload = append(payload, PayloadRule{
			RuleType:   target.RuleType,
			Identifier: target.Identifier,
			Removed:    true,
		})
	}

	return sortedPayloadRules(payload)
}

func fullSyncPayload(targets []Target) []PayloadRule {
	payload := make([]PayloadRule, 0, len(targets))
	for _, target := range targets {
		payload = append(payload, payloadRuleFromTarget(target))
	}
	return sortedPayloadRules(payload)
}

func payloadRuleFromTarget(target Target) PayloadRule {
	return PayloadRule{
		RuleType:      target.RuleType,
		Identifier:    target.Identifier,
		Policy:        target.Policy,
		CELExpression: target.CELExpression,
		CustomMessage: target.CustomMessage,
		CustomURL:     target.CustomURL,
		AppName:       target.AppName,
		PayloadHash:   target.PayloadHash,
	}
}

func targetsEqual(a []Target, b []Target) bool {
	return slices.EqualFunc(sortedTargets(a), sortedTargets(b), func(left Target, right Target) bool {
		return left == right
	})
}

func sortedTargets(targets []Target) []Target {
	out := slices.Clone(targets)
	slices.SortFunc(out, func(left Target, right Target) int {
		if n := cmp.Compare(ruleTypeSort(left.RuleType), ruleTypeSort(right.RuleType)); n != 0 {
			return n
		}
		if n := cmp.Compare(left.Identifier, right.Identifier); n != 0 {
			return n
		}
		return cmp.Compare(left.PayloadHash, right.PayloadHash)
	})
	return out
}

func sortedPayloadRules(rules []PayloadRule) []PayloadRule {
	out := slices.Clone(rules)
	slices.SortFunc(out, func(left PayloadRule, right PayloadRule) int {
		if n := cmp.Compare(ruleTypeSort(left.RuleType), ruleTypeSort(right.RuleType)); n != 0 {
			return n
		}
		if n := cmp.Compare(left.Identifier, right.Identifier); n != 0 {
			return n
		}
		if n := cmp.Compare(left.PayloadHash, right.PayloadHash); n != 0 {
			return n
		}
		switch {
		case left.Removed == right.Removed:
			return 0
		case left.Removed:
			return 1
		default:
			return -1
		}
	})
	return out
}

func ruleTypeSort(ruleType string) int {
	switch ruleType {
	case "cdhash":
		return 1
	case "binary":
		return 2
	case "signingid":
		return 3
	case "certificate":
		return 4
	case "teamid":
		return 5
	default:
		return 6
	}
}

func countTargets(targets []Target) RuleCounts {
	var counts RuleCounts
	for _, target := range targets {
		switch target.RuleType {
		case "binary":
			counts.Binary++
			if target.Policy == "allowlist_compiler" {
				counts.Compiler++
			}
		case "certificate":
			counts.Certificate++
		case "teamid":
			counts.TeamID++
		case "signingid":
			counts.SigningID++
		case "cdhash":
			counts.CDHash++
		}
	}
	return counts
}
