import type { Host } from "@lib/api";

export type HostPrimaryUserSource = NonNullable<Host["primary_user_sources"]>[number];

export function manualPrimaryUserSource(
  sources: readonly HostPrimaryUserSource[] | null | undefined,
): HostPrimaryUserSource | null {
  return (sources ?? []).find((source) => source.source === "manual") ?? null;
}
