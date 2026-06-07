import type { LabelRef, TargetLabel } from "@/lib/api";

export type LabelTargetSet = {
  include: LabelRef[];
  exclude: LabelRef[];
};

export type TargetSummaryInput = LabelTargetSet | TargetLabel[] | null | undefined;

export function emptyLabelTargetSet(): LabelTargetSet {
  return { include: [], exclude: [] };
}

export function normalizeLabelTargetSet(targets: LabelTargetSet | null | undefined): LabelTargetSet {
  return {
    include: targets?.include ?? [],
    exclude: targets?.exclude ?? [],
  };
}

export function targetSetFromFlatTargets(targets: TargetLabel[] | null | undefined): LabelTargetSet {
  const rows = targets ?? [];
  return {
    include: rows.filter((target) => target.effect === "include").map(labelRefFromTarget),
    exclude: rows.filter((target) => target.effect === "exclude").map(labelRefFromTarget),
  };
}

export function flatTargetsFromTargetSet(targets: LabelTargetSet | null | undefined): TargetLabel[] {
  const targetSet = normalizeLabelTargetSet(targets);
  return [
    ...targetSet.include.map((target) => ({ ...target, effect: "include" as const })),
    ...targetSet.exclude.map((target) => ({ ...target, effect: "exclude" as const })),
  ];
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

function labelRefFromTarget(target: TargetLabel): LabelRef {
  return { label_id: target.label_id };
}
