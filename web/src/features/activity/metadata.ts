import type { ActivityEvent } from "@lib/api";

export const ACTIVITY_SCOPE_VALUES = ["user", "system", "all"] as const;

export type ActivityScope = (typeof ACTIVITY_SCOPE_VALUES)[number];

const ACTIVITY_AREA_LABELS = {
  hosts: "Hosts",
  osquery: "osquery",
} as const satisfies Record<ActivityEvent["area"], string>;

const ACTIVITY_ACTION_LABELS = {
  orbit_host_enrolled: "Orbit Host Enrolled",
  osquery_host_enrolled: "osquery Host Enrolled",
  host_deleted: "Host Deleted",
  hosts_deleted: "Hosts Deleted",
  host_inventory_requested: "Host Inventory Requested",
  host_primary_user_set: "Primary User Set",
  host_primary_user_cleared: "Primary User Cleared",
  policy_created: "Policy Created",
  policy_updated: "Policy Updated",
  policy_deleted: "Policy Deleted",
  policies_deleted: "Policies Deleted",
  policy_remediation_requested: "Policy Remediation Requested",
  report_created: "Report Created",
  report_updated: "Report Updated",
  report_deleted: "Report Deleted",
  reports_deleted: "Reports Deleted",
  live_query_started: "Live Query Started",
  live_query_stopped: "Live Query Stopped",
} as const satisfies Record<ActivityEvent["action"], string>;

export const ACTIVITY_AREA_VALUES = keysOf(ACTIVITY_AREA_LABELS);

export const ACTIVITY_ACTION_VALUES = keysOf(ACTIVITY_ACTION_LABELS);

export const ACTIVITY_AREA_OPTIONS = ACTIVITY_AREA_VALUES.map((value) => ({
  value,
  label: ACTIVITY_AREA_LABELS[value],
}));

export const ACTIVITY_ACTION_OPTIONS = ACTIVITY_ACTION_VALUES.map((value) => ({
  value,
  label: ACTIVITY_ACTION_LABELS[value],
}));

function keysOf<const Labels extends Record<string, string>>(labels: Labels) {
  return Object.keys(labels).filter((key): key is Extract<keyof Labels, string> => key in labels);
}
