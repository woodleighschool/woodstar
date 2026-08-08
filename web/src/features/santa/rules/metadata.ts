import type { SantaRule } from "@lib/api";
import { enumLabel, type EnumMetadataMap, enumOptions } from "@lib/enum-metadata";

export type SantaRuleType = SantaRule["rule_type"];
export type SantaRulePolicy = SantaRule["policy"];

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
    description: "SHA-256 hash of the exact binary.",
    variant: "info",
  },
  certificate: {
    name: "Certificate",
    description: "SHA-256 hash of the signing certificate.",
    variant: "success",
  },
  teamid: {
    name: "Team ID",
    description: "10-character Apple Team ID.",
    variant: "warning",
  },
  signingid: {
    name: "Signing ID",
    description: "Signing identifier with team or platform prefix.",
    variant: "outline",
    badgeClassName: "border-violet-500/20 bg-violet-500/10 text-violet-600 dark:text-violet-400",
  },
  cdhash: {
    name: "CDHash",
    description: "Code directory hash of the binary.",
    variant: "error",
  },
  bundle: {
    name: "Bundle",
    description: "A server-side rule expanded to the collected executables in a Santa bundle.",
    variant: "outline",
  },
} satisfies EnumMetadataMap<SantaRuleType>;

export const RULE_TYPE_OPTIONS = enumOptions(RULE_TYPES, RULE_TYPE_VALUES);

export const POLICY_VALUES = [
  "allowlist",
  "allowlist_compiler",
  "blocklist",
  "silent_blocklist",
  "silent_gui_blocklist",
  "silent_tty_blocklist",
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
    description:
      "Allow matching compilers to create transitive rules when that setting is enabled.",
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
  silent_gui_blocklist: {
    name: "Silent GUI Blocklist",
    description: "Block matching software without showing Santa's GUI notification.",
    variant: "error",
  },
  silent_tty_blocklist: {
    name: "Silent TTY Blocklist",
    description: "Block matching software without printing Santa's TTY message.",
    variant: "error",
  },
  cel: {
    name: "CEL",
    description: "Use a Santa CEL expression to decide whether the rule applies.",
    variant: "warning",
  },
} satisfies EnumMetadataMap<SantaRulePolicy>;

export const POLICY_OPTIONS = enumOptions(POLICIES, POLICY_VALUES);

export const RULE_IDENTIFIER_RULES: Record<
  SantaRuleType,
  { pattern: RegExp; placeholder?: string }
> = {
  binary: {
    pattern: /^[0-9a-fA-F]{64}$/,
    placeholder: "fc6679da622c3ff38933220b8e73c7322ecdc94b4570c50ecab0da311b292682",
  },
  certificate: {
    pattern: /^[0-9a-fA-F]{64}$/,
    placeholder: "7ae80b9ab38af0c63a9a81765f434d9a7cd8f720eb6037ef303de39d779bc258",
  },
  bundle: {
    pattern: /^[0-9a-fA-F]{64}$/,
  },
  cdhash: {
    pattern: /^[0-9a-fA-F]{40}$/,
    placeholder: "dbe8c39801f93e05fc7bc53a02af5b4d3cfc670a",
  },
  signingid: {
    pattern: /^(?:[A-Z0-9]{10}|platform):[a-zA-Z0-9.-]+$/,
    placeholder: "UBF8T346G9:com.microsoft.VSCode",
  },
  teamid: {
    pattern: /^[A-Z0-9]{10}$/,
    placeholder: "EQHXZ8M8AV",
  },
};

export function ruleTypeLabel(ruleType: SantaRuleType) {
  return enumLabel(RULE_TYPES, ruleType);
}
