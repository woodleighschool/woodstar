import type { Label } from "@/lib/api";

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
