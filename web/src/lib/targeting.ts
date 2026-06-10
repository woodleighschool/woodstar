import type { LabelRef } from "@/lib/api";

export type LabelTargetSet = {
  include: LabelRef[];
  exclude: LabelRef[];
};

export type FlatLabelTarget = LabelRef & { effect: "include" | "exclude" };
export type TargetSummaryInput = LabelTargetSet | FlatLabelTarget[] | null | undefined;

export function emptyLabelTargetSet(): LabelTargetSet {
  return { include: [], exclude: [] };
}

export function normalizeLabelTargetSet(targets: LabelTargetSet | null | undefined): LabelTargetSet {
  return {
    include: targets?.include ?? [],
    exclude: targets?.exclude ?? [],
  };
}

function targetSetFromFlatTargets(targets: FlatLabelTarget[] | null | undefined): LabelTargetSet {
  const rows = targets ?? [];
  return {
    include: rows.filter((target) => target.effect === "include").map(labelRefFromTarget),
    exclude: rows.filter((target) => target.effect === "exclude").map(labelRefFromTarget),
  };
}

export function targetLabelIDs(targets: TargetSummaryInput) {
  const targetSet = targetSetFromInput(targets);
  return [...targetSet.include, ...targetSet.exclude].map((target) => target.label_id);
}

export function targetSummary(targets: TargetSummaryInput) {
  const targetSet = targetSetFromInput(targets);
  const includeCount = targetSet.include.length;
  const excludeCount = targetSet.exclude.length;
  if (includeCount === 0) return "No targets";
  const include = `${includeCount} include label${includeCount === 1 ? "" : "s"}`;
  if (excludeCount === 0) return include;
  return `${include}, ${excludeCount} excluded`;
}

function targetSetFromInput(targets: TargetSummaryInput): LabelTargetSet {
  if (Array.isArray(targets)) return targetSetFromFlatTargets(targets);
  return normalizeLabelTargetSet(targets);
}

function labelRefFromTarget(target: FlatLabelTarget): LabelRef {
  return { label_id: target.label_id };
}
