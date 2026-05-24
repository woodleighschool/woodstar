export interface LabelChip {
  id: number;
  name: string;
}

export function labelsFromIDs(labelIDs: number[], labelsByID: ReadonlyMap<number, LabelChip>): LabelChip[] {
  return labelIDs.map((id) => labelsByID.get(id) ?? { id, name: `Label ${id}` });
}
