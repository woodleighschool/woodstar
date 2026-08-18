import type { OsqueryPolicyHostStatus, OsqueryPolicyRemediationSummary } from "@lib/api";
import type { EnumMetadataMap, StatusMetadataMap } from "@lib/enum-metadata";

export type PolicyResultStatus = OsqueryPolicyHostStatus["status"];
export type PolicyResultDisplayStatus = PolicyResultStatus | "stopped";
export type RemediationRunStatus = NonNullable<OsqueryPolicyHostStatus["remediation"]>["status"];
export type RemediationStatusFilter = RemediationRunStatus | "not_run";
export type PolicyRemediationMode = "none" | "manual" | "automatic";

export const POLICY_RESULT_STATUS_VALUES = ["pending", "pass", "fail", "error"] as const;

export const POLICY_RESULT_STATUSES = {
  pending: {
    name: "Pending",
    variant: "default",
  },
  pass: {
    name: "Passing",
    variant: "success",
  },
  fail: {
    name: "Failing",
    variant: "error",
  },
  error: {
    name: "Error",
    variant: "warning",
  },
  stopped: {
    name: "Stopped",
    variant: "default",
  },
} satisfies StatusMetadataMap<PolicyResultDisplayStatus>;

export const POLICY_RESULT_STATUS_OPTIONS = [
  { label: "Pending", value: "pending" },
  { label: "Passing", value: "pass" },
  { label: "Failing", value: "fail" },
  { label: "Error", value: "error" },
] satisfies { label: string; value: PolicyResultStatus }[];

export const POLICY_REMEDIATION_MODES = {
  none: {
    name: "None",
    description: "No remediation script is configured.",
  },
  manual: {
    name: "Manual",
    description: "A remediation script is configured for manual runs.",
  },
  automatic: {
    name: "Automatic",
    description: "A remediation script runs when an eligible host newly becomes failing.",
  },
} satisfies EnumMetadataMap<PolicyRemediationMode>;

export function policyRemediationMode(
  remediation: OsqueryPolicyRemediationSummary,
): PolicyRemediationMode {
  if (!remediation.configured) return "none";
  return remediation.automatic ? "automatic" : "manual";
}

export const REMEDIATION_RUN_STATUSES = {
  queued: {
    name: "Queued",
    variant: "default",
  },
  in_progress: {
    name: "In progress",
    variant: "info",
  },
  succeeded: {
    name: "Succeeded",
    variant: "success",
  },
  failed: {
    name: "Failed",
    variant: "error",
  },
  no_response: {
    name: "No response",
    variant: "warning",
  },
  cancelled: {
    name: "Cancelled",
    variant: "default",
  },
} satisfies StatusMetadataMap<RemediationRunStatus>;

export const REMEDIATION_STATUS_FILTER_VALUES = [
  "failed",
  "no_response",
  "in_progress",
  "queued",
  "succeeded",
  "cancelled",
  "not_run",
] as const satisfies readonly RemediationStatusFilter[];

export const REMEDIATION_STATUS_FILTER_OPTIONS = [
  ...REMEDIATION_STATUS_FILTER_VALUES.filter((status) => status !== "not_run").map((status) => ({
    label: REMEDIATION_RUN_STATUSES[status].name,
    value: status,
  })),
  { label: "Not run", value: "not_run" },
] satisfies { label: string; value: RemediationStatusFilter }[];
