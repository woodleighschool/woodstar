import type { SantaRuleMutation } from "@/hooks/use-santa";

export const RULE_TYPES = [
  { value: "binary", label: "Binary" },
  { value: "certificate", label: "Certificate" },
  { value: "teamid", label: "Team ID" },
  { value: "signingid", label: "Signing ID" },
  { value: "cdhash", label: "CDHash" },
] as const satisfies readonly { value: SantaRuleMutation["rule_type"]; label: string }[];

export type RuleType = (typeof RULE_TYPES)[number]["value"];

export const RULE_IDENTIFIER_RULES: Record<RuleType, { pattern: RegExp; hint: string }> = {
  binary: {
    pattern: /^[0-9a-fA-F]{64}$/,
    hint: "Use a 64 character SHA-256 hex hash.",
  },
  certificate: {
    pattern: /^[0-9a-fA-F]{64}$/,
    hint: "Use a 64 character certificate SHA-256 hex fingerprint.",
  },
  cdhash: {
    pattern: /^[0-9a-fA-F]{40}$/,
    hint: "Use a 40 character CDHash hex value.",
  },
  signingid: {
    pattern: /^(?:[A-Z0-9]{10}|platform):[a-zA-Z0-9.-]+$/,
    hint: "Use TEAMID:bundle.identifier or platform:bundle.identifier.",
  },
  teamid: {
    pattern: /^[A-Z0-9]{10}$/,
    hint: "Use a 10 character uppercase Team ID.",
  },
};

export function ruleTypeLabel(ruleType: string) {
  return RULE_TYPES.find((type) => type.value === ruleType)?.label ?? ruleType;
}
