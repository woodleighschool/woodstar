import type { Host } from "@/hooks/use-hosts";

export type HostDeviceMapping = NonNullable<Host["device_mappings"]>[number];

const SOURCE_PRIORITY: Record<string, number> = {
  manual: 0,
  orbit_profile: 1,
  santa_primary_user: 1,
};

export function sortDeviceMappings(mappings: readonly HostDeviceMapping[] | null | undefined): HostDeviceMapping[] {
  return [...(mappings ?? [])].sort((a, b) => {
    const byPriority = sourcePriority(a.source) - sourcePriority(b.source);
    if (byPriority !== 0) return byPriority;
    return a.source.localeCompare(b.source) || a.email.localeCompare(b.email);
  });
}

export function primaryDeviceMapping(
  mappings: readonly HostDeviceMapping[] | null | undefined,
): HostDeviceMapping | null {
  return sortDeviceMappings(mappings)[0] ?? null;
}

export function manualDeviceMapping(
  mappings: readonly HostDeviceMapping[] | null | undefined,
): HostDeviceMapping | null {
  return (mappings ?? []).find((mapping) => mapping.source === "manual") ?? null;
}

export function secondaryDeviceMappings(mappings: readonly HostDeviceMapping[] | null | undefined) {
  const sorted = sortDeviceMappings(mappings);
  return sorted.slice(1);
}

function sourcePriority(source: string) {
  return SOURCE_PRIORITY[source] ?? 10;
}
