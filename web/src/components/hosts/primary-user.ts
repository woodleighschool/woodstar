import type { Host } from "@/lib/api";

export type HostPrimaryUserSource = NonNullable<Host["primary_user_sources"]>[number];

const SOURCE_PRIORITY: Record<string, number> = {
  manual: 0,
  orbit_profile: 1,
};

export function sortPrimaryUserSources(
  sources: readonly HostPrimaryUserSource[] | null | undefined,
): HostPrimaryUserSource[] {
  return [...(sources ?? [])].toSorted((a, b) => {
    const byPriority = sourcePriority(a.source) - sourcePriority(b.source);
    if (byPriority !== 0) return byPriority;
    return a.source.localeCompare(b.source) || a.email.localeCompare(b.email);
  });
}

export function primaryUserSource(
  sources: readonly HostPrimaryUserSource[] | null | undefined,
): HostPrimaryUserSource | null {
  return sortPrimaryUserSources(sources)[0] ?? null;
}

export function manualPrimaryUserSource(
  sources: readonly HostPrimaryUserSource[] | null | undefined,
): HostPrimaryUserSource | null {
  return (sources ?? []).find((source) => source.source === "manual") ?? null;
}

export function secondaryPrimaryUserSources(
  sources: readonly HostPrimaryUserSource[] | null | undefined,
) {
  const sorted = sortPrimaryUserSources(sources);
  return sorted.slice(1);
}

function sourcePriority(source: string) {
  return SOURCE_PRIORITY[source] ?? 10;
}
