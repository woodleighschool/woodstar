import type { OsqueryCheckHostStatus } from "@lib/api";
import type { StatusMetadataMap } from "@lib/enum-metadata";

export type CheckResultStatus = NonNullable<OsqueryCheckHostStatus["response"]> | "not_run";

export const CHECK_RESULT_STATUSES = {
  pass: {
    name: "Passing",
    variant: "success",
  },
  fail: {
    name: "Failing",
    variant: "error",
  },
  not_run: {
    name: "Not Run",
    variant: "default",
  },
} satisfies StatusMetadataMap<CheckResultStatus>;

export function checkResultStatus(
  response: OsqueryCheckHostStatus["response"] | null | undefined,
): CheckResultStatus {
  return response ?? "not_run";
}
