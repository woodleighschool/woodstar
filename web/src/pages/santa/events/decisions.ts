import type { SantaEventDecisionFilter } from "@/hooks/use-santa-events";
import type { SantaExecutionEvent as SantaEvent, SantaFileAccessEvent } from "@/lib/api";
import { enumOptions, type StatusMetadataMap } from "@/lib/enum-metadata";

export type SantaExecutionDecision = SantaEvent["decision"];
export type SantaFileAccessDecision = SantaFileAccessEvent["decision"];

export const EXECUTION_DECISIONS = {
  unknown: {
    name: "Unknown",
    description: "Santa reported an execution event without a specific decision.",
    variant: "default",
  },
  allow_unknown: {
    name: "Allow Unknown",
    description: "Allowed because no rule matched while the client was in Monitor mode.",
    variant: "success",
  },
  allow_binary: {
    name: "Allow Binary",
    description: "Allowed by a rule for this exact binary.",
    variant: "success",
  },
  allow_certificate: {
    name: "Allow Certificate",
    description: "Allowed by a matching signing certificate.",
    variant: "success",
  },
  allow_scope: {
    name: "Allow Scope",
    description: "Allowed by an approved path or script exception.",
    variant: "success",
  },
  allow_teamid: {
    name: "Allow Team ID",
    description: "Allowed by a matching Team ID rule.",
    variant: "success",
  },
  allow_signingid: {
    name: "Allow Signing ID",
    description: "Allowed by a matching Signing ID rule.",
    variant: "success",
  },
  allow_cdhash: {
    name: "Allow CDHash",
    description: "Allowed by a matching CDHash rule.",
    variant: "success",
  },
  allow_platform: {
    name: "Allow Platform",
    description: "Allowed because macOS reported the binary as platform-trusted.",
    variant: "success",
  },
  block_unknown: {
    name: "Block Unknown",
    description: "Blocked because no rule matched while the client was in Lockdown mode.",
    variant: "error",
  },
  block_binary: {
    name: "Block Binary",
    description: "Blocked by a rule for this exact binary.",
    variant: "error",
  },
  block_certificate: {
    name: "Block Certificate",
    description: "Blocked by a matching signing certificate.",
    variant: "error",
  },
  block_scope: {
    name: "Block Scope",
    description: "Blocked by a blocked path rule or Page Zero protection.",
    variant: "error",
  },
  block_teamid: {
    name: "Block Team ID",
    description: "Blocked by a matching Team ID rule.",
    variant: "error",
  },
  block_signingid: {
    name: "Block Signing ID",
    description: "Blocked by a matching Signing ID rule.",
    variant: "error",
  },
  block_cdhash: {
    name: "Block CDHash",
    description: "Blocked by a matching CDHash rule.",
    variant: "error",
  },
  block_binary_mismatch: {
    name: "Block Binary Mismatch",
    description: "Blocked because the kernel and disk binary hashes did not match.",
    variant: "error",
  },
  bundle_binary: {
    name: "Bundle Binary",
    description: "Metadata for a binary inside a bundle. It is not an allow or block decision.",
    variant: "default",
  },
} satisfies StatusMetadataMap<SantaExecutionDecision>;

export const DECISION_FILTER_VALUES = [
  "allowed",
  "blocked",
  "unknown",
  "allow_unknown",
  "allow_binary",
  "allow_certificate",
  "allow_scope",
  "allow_teamid",
  "allow_signingid",
  "allow_cdhash",
  "allow_platform",
  "block_unknown",
  "block_binary",
  "block_certificate",
  "block_scope",
  "block_teamid",
  "block_signingid",
  "block_cdhash",
  "block_binary_mismatch",
  "bundle_binary",
] as const satisfies readonly SantaEventDecisionFilter[];

export const DECISION_FILTERS = [
  {
    value: "allowed",
    label: "Allowed",
  },
  {
    value: "blocked",
    label: "Blocked",
  },
  ...enumOptions(EXECUTION_DECISIONS),
] satisfies Array<{ value: SantaEventDecisionFilter; label: string }>;

export const FILE_ACCESS_DECISIONS = {
  unknown: {
    name: "Unknown",
    description: "Santa reported a file access event without a specific decision.",
    variant: "default",
  },
  denied: {
    name: "Denied",
    description: "Santa denied the file access because the rule matched and access was blocked.",
    variant: "error",
  },
  denied_invalid_signature: {
    name: "Denied (Invalid Signature)",
    description:
      "Santa denied the file access because the accessing process had an invalid signature.",
    variant: "error",
  },
  audit_only: {
    name: "Audit Only",
    description: "Santa recorded the file access event without blocking it.",
    variant: "success",
  },
} satisfies StatusMetadataMap<SantaFileAccessDecision>;

export const FILE_ACCESS_DECISION_VALUES = [
  "unknown",
  "denied",
  "denied_invalid_signature",
  "audit_only",
] as const satisfies readonly SantaFileAccessDecision[];

export const FILE_ACCESS_DECISION_FILTERS = enumOptions(FILE_ACCESS_DECISIONS);

export function fileName(path: string) {
  const parts = path.split("/").filter(Boolean);
  return parts.at(-1) ?? "";
}

export function executableLabel(event: SantaEvent) {
  return event.executable.file_name || fileName(event.file_path) || event.executable.sha256;
}

export function fileAccessEventLabel(event: SantaFileAccessEvent) {
  return fileName(event.target) || event.target;
}
