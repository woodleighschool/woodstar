import type { OsqueryPolicyHostStatus } from "@lib/api";
import type { StatusMetadataMap } from "@lib/enum-metadata";

export type PolicyResultStatus = OsqueryPolicyHostStatus["status"];
export type PolicyResultDisplayStatus = PolicyResultStatus | "stopped";
export type RemediationRunStatus = NonNullable<OsqueryPolicyHostStatus["remediation"]>["status"];

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
