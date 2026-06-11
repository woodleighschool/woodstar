import type { SantaRulePolicy, SantaRuleType } from "@/hooks/use-santa-rules";
import { enumLabel, enumOptions, type EnumMetadataMap, type StatusMetadataMap } from "@/lib/enum-metadata";

export const RULE_TYPE_VALUES = [
  "binary",
  "certificate",
  "teamid",
  "signingid",
  "cdhash",
  "bundle",
] as const satisfies readonly SantaRuleType[];

export const RULE_TYPES = {
  binary: {
    name: "Binary",
    description: "A rule keyed to one executable SHA-256 hash.",
  },
  certificate: {
    name: "Certificate",
    description: "A rule keyed to a signing certificate SHA-256 fingerprint.",
  },
  teamid: {
    name: "Team ID",
    description: "A rule keyed to an Apple Developer Team ID.",
  },
  signingid: {
    name: "Signing ID",
    description: "A rule keyed to a Team ID and bundle identifier pair.",
  },
  cdhash: {
    name: "CDHash",
    description: "A rule keyed to a Mach-O code directory hash.",
  },
  bundle: {
    name: "Bundle",
    description: "A server-side rule expanded to the collected executables in a Santa bundle.",
  },
} satisfies EnumMetadataMap<SantaRuleType>;

export const RULE_TYPE_OPTIONS = enumOptions(RULE_TYPES);

export const POLICY_VALUES = [
  "allowlist",
  "allowlist_compiler",
  "blocklist",
  "silent_blocklist",
  "cel",
] as const satisfies readonly SantaRulePolicy[];

export const POLICIES = {
  allowlist: {
    name: "Allowlist",
    description: "Allow matching software on targeted hosts.",
    variant: "success",
  },
  allowlist_compiler: {
    name: "Compiler Allowlist",
    description: "Allow matching compilers to create transitive rules when that setting is enabled.",
    variant: "success",
  },
  blocklist: {
    name: "Blocklist",
    description: "Block matching software on targeted hosts.",
    variant: "error",
  },
  silent_blocklist: {
    name: "Silent Blocklist",
    description: "Block matching software without showing a custom user-facing Santa message.",
    variant: "error",
  },
  cel: {
    name: "CEL",
    description: "Use a Santa CEL expression to decide whether the rule applies.",
    variant: "default",
  },
} satisfies StatusMetadataMap<SantaRulePolicy>;

export const POLICY_OPTIONS = enumOptions(POLICIES);

export const RULE_IDENTIFIER_RULES: Record<SantaRuleType, { pattern: RegExp; hint: string }> = {
  binary: {
    pattern: /^[0-9a-fA-F]{64}$/,
    hint: "Use a 64 character SHA-256 hex hash.",
  },
  certificate: {
    pattern: /^[0-9a-fA-F]{64}$/,
    hint: "Use a 64 character certificate SHA-256 hex fingerprint.",
  },
  bundle: {
    pattern: /^[0-9a-fA-F]{64}$/,
    hint: "Pick a fully collected Santa bundle.",
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
  return enumLabel(RULE_TYPES, ruleType);
}

export function policyLabel(policy: string) {
  return enumLabel(POLICIES, policy);
}
