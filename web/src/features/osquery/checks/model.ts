import type { OsqueryCheckHostStatus } from "@lib/api";
import type { StatusMetadataMap } from "@lib/enum-metadata";

export type CheckResultStatus = OsqueryCheckHostStatus["status"];
export type CheckResultDisplayStatus = CheckResultStatus | "error" | "stopped";

export const CHECK_RESULT_STATUS_VALUES = ["pass", "fail", "pending"] as const;

export const CHECK_RESULT_STATUSES = {
  pass: {
    name: "Passing",
    variant: "success",
  },
  fail: {
    name: "Failing",
    variant: "error",
  },
  pending: {
    name: "Pending",
    variant: "default",
  },
  error: {
    name: "Error",
    variant: "error",
  },
  stopped: {
    name: "Stopped",
    variant: "default",
  },
} satisfies StatusMetadataMap<CheckResultDisplayStatus>;

export const CHECK_RESULT_STATUS_OPTIONS = [
  { label: "Passing", value: "pass" },
  { label: "Failing", value: "fail" },
  { label: "Pending", value: "pending" },
] satisfies { label: string; value: CheckResultStatus }[];
