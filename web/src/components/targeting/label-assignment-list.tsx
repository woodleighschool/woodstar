import { revalidateLogic, useForm } from "@tanstack/react-form";
import { MoreHorizontal, Plus } from "lucide-react";
import { useMemo, useState } from "react";
import { z } from "zod";

import { TableSurface } from "@components/data-table/table-surface";
import { encodeSort } from "@components/data-table/use-data-table-search";
import { FormActions } from "@components/form-actions";
import { focusFirstInvalidField } from "@components/form-tabs";
import { PanelEmptyState } from "@components/panel-empty-state";
import { TargetSection } from "@components/targeting/target-section";
import { Button } from "@components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { ValidatedFormField } from "@components/validated-form-field";
import { LabelPicker } from "@features/labels/components/label-picker";
import { useLabels } from "@features/labels/queries";
import type { LabelRef } from "@lib/api";
import { MAX_PAGE_SIZE } from "@lib/pagination";
const labelAssignmentSchema = z.object({
  label_id: z
    .number()
    .int("Label selection is invalid.")
    .positive("Pick a label.")
    .nullable()
    .refine((value) => value !== null, "Pick a label."),
});
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
  return (
    <TargetSection
      title={title}
      action={
        <Button type="button" variant="outline" size="sm" onClick={() => setAdding(true)}>
          <Plus data-icon="inline-start" />
          {addLabel}
        </Button>
      }
    >
      {rows.length > 0 ? (
        <TableSurface>
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
                      onRemove={() =>
                        onChange(rows.filter((item) => item.label_id !== row.label_id))
                      }
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableSurface>
      ) : (
        <PanelEmptyState>{emptyText}</PanelEmptyState>
      )}

      {adding ? (
        <LabelAssignmentDialog
          title={addLabel}
          unavailableLabelIDs={unavailableLabelIDs}
          includeBuiltins={includeBuiltins}
          onClose={() => setAdding(false)}
          onSave={(labelID) => {
            onChange([...rows, { label_id: labelID }]);
            setAdding(false);
          }}
        />
      ) : null}
    </TargetSection>
  );
}
function LabelAssignmentDialog({
  title,
  unavailableLabelIDs,
  includeBuiltins,
  onClose,
  onSave,
}: {
  title: string;
  unavailableLabelIDs: readonly number[];
  includeBuiltins: boolean;
  onClose: () => void;
  onSave: (labelID: number) => void;
}) {
  const form = useForm({
    defaultValues: { label_id: null as number | null },
    validationLogic: revalidateLogic({
      mode: "submit",
      modeAfterSubmission: "change",
    }),
    validators: { onDynamic: labelAssignmentSchema },
    onSubmit: ({ value }) => {
      const assignment = labelAssignmentSchema.parse(value);
      if (assignment.label_id === null) return;
      onSave(assignment.label_id);
    },
  });
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent>
        <div className="contents">
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
          </DialogHeader>
          <form.Field name="label_id">
            {(field) => (
              <ValidatedFormField field={field} label="Label" required>
                {(control) => (
                  <LabelPicker
                    value={field.state.value === null ? [] : [field.state.value]}
                    onChange={(ids) => field.handleChange(ids[0] ?? null)}
                    selectionMode="single"
                    includeBuiltins={includeBuiltins}
                    unavailableLabelIDs={unavailableLabelIDs}
                    required
                    invalid={control["aria-invalid"]}
                    placeholder="Select Label"
                  />
                )}
              </ValidatedFormField>
            )}
          </form.Field>
          <FormActions
            form={form}
            submitLabel="Add"
            onSubmit={async () => {
              await form.handleSubmit();
              if (!form.state.isValid) focusFirstInvalidField();
            }}
            onCancel={onClose}
            className="justify-end"
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
function LabelAssignmentRowActions({ onRemove }: { onRemove: () => void }) {
  return (
    <div className="flex justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button type="button" variant="ghost" size="icon-sm" />}>
          <MoreHorizontal />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-36">
          <DropdownMenuGroup>
            <DropdownMenuItem variant="destructive" onClick={onRemove}>
              Delete
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
