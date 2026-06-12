import type { Host } from "@/hooks/use-hosts";

export type HostUserAffinityMapping = NonNullable<Host["user_affinity"]["mappings"]>[number];

const SOURCE_PRIORITY: Record<string, number> = {
  manual: 0,
  orbit_profile: 1,
  santa_primary_user: 1,
};

export function sortUserAffinityMappings(
  mappings: readonly HostUserAffinityMapping[] | null | undefined,
): HostUserAffinityMapping[] {
  return [...(mappings ?? [])].toSorted((a, b) => {
    const byPriority = sourcePriority(a.source) - sourcePriority(b.source);
    if (byPriority !== 0) return byPriority;
    return a.source.localeCompare(b.source) || a.email.localeCompare(b.email);
  });
}

export function primaryUserAffinityMapping(
  mappings: readonly HostUserAffinityMapping[] | null | undefined,
): HostUserAffinityMapping | null {
  return sortUserAffinityMappings(mappings)[0] ?? null;
}

export function manualUserAffinityMapping(
  mappings: readonly HostUserAffinityMapping[] | null | undefined,
): HostUserAffinityMapping | null {
  return (mappings ?? []).find((mapping) => mapping.source === "manual") ?? null;
}

export function secondaryUserAffinityMappings(
  mappings: readonly HostUserAffinityMapping[] | null | undefined,
) {
  const sorted = sortUserAffinityMappings(mappings);
  return sorted.slice(1);
}

function sourcePriority(source: string) {
  return SOURCE_PRIORITY[source] ?? 10;
}
