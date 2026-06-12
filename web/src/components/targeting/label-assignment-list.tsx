import type { ColumnDef } from "@tanstack/react-table";
import { Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { EmptyPanel } from "@/components/empty-panel";
import { AssignmentLabelField } from "@/components/targeting/assignment-label-field";
import { TargetSection } from "@/components/targeting/target-section";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { encodeSort, MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import { useLabels } from "@/hooks/use-labels";
import type { LabelRef } from "@/lib/api";

export function LabelAssignmentList({
  title,
  addLabel,
  emptyText,
  rows,
  crossListLabelIDs = [],
  includeBuiltins = true,
  onChange,
}: {
  title: string;
  addLabel: string;
  emptyText: string;
  rows: LabelRef[];
  crossListLabelIDs?: readonly number[];
  includeBuiltins?: boolean;
  onChange: (rows: LabelRef[]) => void;
}) {
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState<number | null>(null);
  const labels = useLabels({
    per_page: MAX_PAGE_SIZE,
    sort: encodeSort("name"),
    label_type: includeBuiltins ? undefined : "regular",
  });
  const labelsByID = useMemo(
    () =>
      new Map<number, string>((labels.data?.items ?? []).map((label) => [label.id, label.name])),
    [labels.data],
  );
  const unavailableLabelIDs = [...rows.map((row) => row.label_id), ...crossListLabelIDs];

  function addRow() {
    if (draft === null) return;
    onChange([...rows, { label_id: draft }]);
    setDraft(null);
    setAdding(false);
  }

  const columns = useMemo<ColumnDef<LabelRef>[]>(
    () => [
      {
        id: "label",
        header: "Label",
        enableSorting: false,
        cell: ({ row }) =>
          labelsByID.get(row.original.label_id) ?? `Label ${row.original.label_id}`,
      },
      {
        id: "actions",
        header: () => null,
        enableSorting: false,
        cell: ({ row }) => (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={`Remove ${title.toLowerCase()}`}
            onClick={() => onChange(rows.filter((item) => item.label_id !== row.original.label_id))}
          >
            <Trash2 />
          </Button>
        ),
      },
    ],
    [labelsByID, onChange, rows, title],
  );

  return (
    <TargetSection
      title={title}
      action={
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => {
            setDraft(null);
            setAdding(true);
          }}
        >
          <Plus data-icon="inline-start" />
          {addLabel}
        </Button>
      }
    >
      {rows.length > 0 ? (
        <DataTableStatic columns={columns} data={rows} />
      ) : (
        <EmptyPanel>{emptyText}</EmptyPanel>
      )}

      <Dialog open={adding} onOpenChange={setAdding}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{addLabel}</DialogTitle>
          </DialogHeader>
          <AssignmentLabelField
            value={draft}
            onChange={setDraft}
            unavailableLabelIDs={unavailableLabelIDs}
            includeBuiltins={includeBuiltins}
          />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setAdding(false)}>
              Close
            </Button>
            <Button type="button" onClick={addRow} disabled={draft === null}>
              Add
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </TargetSection>
  );
}
