export interface LabelTargetRow {
  id: number;
  label_id: number | null;
}

export function selectedLabelTargetIDs(rows: LabelTargetRow[]) {
  return rows.flatMap((row) => (row.label_id === null ? [] : [row.label_id]));
}

export function unavailableLabelTargetIDs(
  rows: LabelTargetRow[],
  excludeLabelIDs: readonly number[],
  currentRowID: number,
) {
  return [
    ...excludeLabelIDs,
    ...rows.flatMap((row) => {
      if (row.id === currentRowID || row.label_id === null) return [];
      return [row.label_id];
    }),
  ];
}
