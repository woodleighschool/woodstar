import type { Schemas } from "@/lib/api";

type LabelScope = Schemas["LabelScopeBody"];

export function targetSummary(scope: LabelScope | undefined, platform?: string | null) {
  const labels = scope?.label_ids?.length ?? 0;
  const platformText = platform ? platformLabel(platform) : "all platforms";
  if (!scope?.mode || labels === 0) return `All hosts, ${platformText}`;

  const labelText = `${labels} label${labels === 1 ? "" : "s"}`;
  switch (scope.mode) {
    case "include_any":
      return `Any of ${labelText}, ${platformText}`;
    case "include_all":
      return `All ${labelText}, ${platformText}`;
    case "exclude_any":
      return `Excluding ${labelText}, ${platformText}`;
    default:
      return `All hosts, ${platformText}`;
  }
}

export function platformLabel(platform?: string | null) {
  if (!platform) return "all platforms";
  if (platform === "darwin") return "macOS";
  return platform;
}
