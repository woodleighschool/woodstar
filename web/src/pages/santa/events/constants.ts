import type { SantaEvent } from "@/hooks/use-santa";

export const DECISION_FILTERS = [
  { value: "allowed", label: "Allowed" },
  { value: "blocked", label: "Blocked" },
  { value: "allow_unknown", label: "Allow Unknown" },
  { value: "allow_binary", label: "Allow Binary" },
  { value: "allow_certificate", label: "Allow Certificate" },
  { value: "allow_scope", label: "Allow Scope" },
  { value: "allow_teamid", label: "Allow Team ID" },
  { value: "allow_signingid", label: "Allow Signing ID" },
  { value: "allow_cdhash", label: "Allow CDHash" },
  { value: "block_unknown", label: "Block Unknown" },
  { value: "block_binary", label: "Block Binary" },
  { value: "block_certificate", label: "Block Certificate" },
  { value: "block_scope", label: "Block Scope" },
  { value: "block_teamid", label: "Block Team ID" },
  { value: "block_signingid", label: "Block Signing ID" },
  { value: "block_cdhash", label: "Block CDHash" },
  { value: "bundle_binary", label: "Bundle Binary" },
  { value: "unknown", label: "Unknown" },
] as const;

export const FILE_ACCESS_DECISION_FILTERS = [
  { value: "denied", label: "Denied" },
  { value: "denied_invalid_signature", label: "Denied Invalid Signature" },
  { value: "audit_only", label: "Audit Only" },
  { value: "unknown", label: "Unknown" },
] as const;

export function fileName(path: string) {
  const parts = path.split("/").filter(Boolean);
  return parts.at(-1) ?? "";
}

export function executableLabel(event: SantaEvent) {
  return event.executable.file_name || fileName(event.file_path) || event.executable.sha256;
}
