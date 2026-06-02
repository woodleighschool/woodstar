import type { TargetLabelRow } from "@/components/targeting/target-label-row-editor";

export function targetLabelSummary(targets: TargetLabelRow[] | null | undefined) {
  const rows = targets ?? [];
  const includeCount = rows.filter((target) => target.effect === "include").length;
  const excludeCount = rows.filter((target) => target.effect === "exclude").length;
  if (includeCount === 0) return "No targets";
  const include = `${includeCount} include label${includeCount === 1 ? "" : "s"}`;
  if (excludeCount === 0) return include;
  return `${include}, ${excludeCount} excluded`;
}
