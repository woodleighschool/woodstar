import type { Label } from "@/hooks/use-labels";

export interface LabelChip {
  id: number;
  name: string;
  builtin_key?: Label["builtin_key"];
}

export function labelsFromIDs(
  labelIDs: number[],
  labelsByID: ReadonlyMap<number, LabelChip>,
): LabelChip[] {
  return labelIDs.map((id) => labelsByID.get(id) ?? { id, name: `Label ${id}` });
}
