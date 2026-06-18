import { MoreHorizontal, Plus } from "lucide-react";
import { useMemo, useState } from "react";

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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { encodeSort } from "@/hooks/use-data-table-search";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
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
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Label</TableHead>
                <TableHead className="w-12">
                  <span className="sr-only">Actions</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.label_id}>
                  <TableCell>{labelsByID.get(row.label_id) ?? `Label ${row.label_id}`}</TableCell>
                  <TableCell className="w-12">
                    <LabelAssignmentRowActions
                      title={title}
                      onRemove={() =>
                        onChange(rows.filter((item) => item.label_id !== row.label_id))
                      }
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
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

function LabelAssignmentRowActions({ title, onRemove }: { title: string; onRemove: () => void }) {
  return (
    <div className="flex justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={`Open ${title.toLowerCase()} actions`}
          >
            <MoreHorizontal />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-36">
          <DropdownMenuGroup>
            <DropdownMenuItem variant="destructive" onSelect={onRemove}>
              Delete
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
