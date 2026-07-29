import type { OsqueryCheckHostStatus } from "@lib/api";
import type { StatusMetadataMap } from "@lib/enum-metadata";

export type CheckResultStatus = OsqueryCheckHostStatus["status"];

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
} satisfies StatusMetadataMap<CheckResultStatus>;

export const CHECK_RESULT_STATUS_OPTIONS = [
  { label: "Passing", value: "pass" },
  { label: "Failing", value: "fail" },
  { label: "Pending", value: "pending" },
] satisfies { label: string; value: CheckResultStatus }[];

export function parseCheckResultStatus(value: unknown): CheckResultStatus | undefined {
  if (typeof value !== "string") return undefined;
  return CHECK_RESULT_STATUS_OPTIONS.find((option) => option.value === value)?.value;
}
