import type { Schemas } from "@/lib/api";

type LabelScope = Schemas["LabelScope"];

export function targetSummary(scope: LabelScope | undefined, platform?: string | null) {
  const platformText = platform ? platformLabel(platform) : "all platforms";
  return `${targetScopeLabel(scope)}, ${platformText}`;
}

export function targetScopeLabel(scope: LabelScope | undefined) {
  const labels = scope?.label_ids?.length ?? 0;
  if (!scope?.mode || labels === 0) return "All hosts";
  const labelText = `${labels} label${labels === 1 ? "" : "s"}`;
  switch (scope.mode) {
    case "include_any":
      return `Any of ${labelText}`;
    case "include_all":
      return `All ${labelText}`;
    case "exclude_any":
      return `Excluding ${labelText}`;
    default:
      return "All hosts";
  }
}

export function platformLabel(platform?: string | null) {
  if (!platform) return "all platforms";
  const labels = platform
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
    .map(platformDisplayLabel);
  if (labels.length === 0) return "all platforms";
  return labels.join(", ");
}

export const QUERYABLE_PLATFORMS = ["darwin", "windows", "linux", "chrome"] as const;

export type QueryablePlatform = (typeof QUERYABLE_PLATFORMS)[number];

export const PLATFORM_LABELS: Record<QueryablePlatform, string> = {
  darwin: "macOS",
  windows: "Windows",
  linux: "Linux",
  chrome: "ChromeOS",
};

const PLATFORM_DISPLAY_LABELS: Partial<Record<string, string>> = {
  ...PLATFORM_LABELS,
  macos: "macOS",
  rhel: "Linux",
  centos: "Linux",
  ubuntu: "Linux",
};

export function platformsFromValue(value?: string | null): QueryablePlatform[] {
  if (!value) return [];
  return value
    .split(",")
    .map((item) => item.trim())
    .filter((item): item is QueryablePlatform => isQueryablePlatform(item));
}

export function platformsToValue(platforms: QueryablePlatform[]) {
  return platforms.length ? platforms.join(",") : undefined;
}

export function isQueryablePlatform(platform: string): platform is QueryablePlatform {
  return (QUERYABLE_PLATFORMS as readonly string[]).includes(platform);
}

export function platformDisplayLabel(platform: string) {
  return PLATFORM_DISPLAY_LABELS[platform] ?? platform;
}
