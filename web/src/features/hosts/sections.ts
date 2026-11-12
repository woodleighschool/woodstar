import type { Host } from "@lib/api";

export const hostSections = [
  { value: "details", label: "Details", path: "/hosts/$id", permissions: [] },
  {
    value: "activity",
    label: "Activity",
    path: "/hosts/$id/activity",
    permissions: [{ resource: "activity", access: "view" }],
  },
  {
    value: "software",
    label: "Software",
    path: "/hosts/$id/software",
    heartbeatSource: "osquery",
    permissions: [{ resource: "software", access: "view" }],
  },
  {
    value: "reports",
    label: "Reports",
    path: "/hosts/$id/reports",
    heartbeatSource: "osquery",
    permissions: [{ resource: "osquery.reports", access: "view" }],
  },
  {
    value: "policies",
    label: "Policies",
    path: "/hosts/$id/policies",
    heartbeatSource: "osquery",
    permissions: [{ resource: "osquery.policies", access: "view" }],
  },
  {
    value: "munki",
    label: "Munki",
    path: "/hosts/$id/munki",
    heartbeatSource: "munki",
    permissions: [{ resource: "munki.software", access: "view" }],
  },
  {
    value: "santa",
    label: "Santa",
    path: "/hosts/$id/santa",
    heartbeatSource: "santa",
    permissions: [
      { resource: "santa.configurations", access: "view" },
      { resource: "santa.rules", access: "view" },
    ],
  },
] as const;

export type HostSection = (typeof hostSections)[number];

export function availableHostSections(host: Pick<Host, "heartbeats">) {
  return hostSections.filter(
    (section) =>
      !("heartbeatSource" in section) ||
      host.heartbeats.some((heartbeat) => heartbeat.source === section.heartbeatSource),
  );
}
