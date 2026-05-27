import type { Schemas } from "@/lib/api";

type LabelScope = Schemas["LabelScope"];

export function targetSummary(scope: LabelScope | undefined) {
  return targetScopeLabel(scope);
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
