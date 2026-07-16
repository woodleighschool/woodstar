import { revalidateLogic, useForm } from "@tanstack/react-form";
import { MoreHorizontal, Plus } from "lucide-react";
import { useMemo, useState } from "react";
import { z } from "zod";

import { EmptyPanel } from "@/components/empty-panel";
import { FormActions } from "@/components/form-actions";
import { FormField } from "@/components/form-field";
import { focusFirstInvalidField } from "@/components/form-tabs";
import { LabelPicker } from "@/components/labels/label-picker";
import { TargetSection } from "@/components/targeting/target-section";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
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
import { useFormExitGuard } from "@/hooks/use-form-exit-guard";
import { useLabels } from "@/hooks/use-labels";
import type { LabelRef } from "@/lib/api";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

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
  const exitGuard = useFormExitGuard({
    form,
    onDiscard: onClose,
    blockNavigation: false,
  });

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) exitGuard.requestDiscard();
      }}
    >
      <DialogContent>
        <form
          noValidate
          className="contents"
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void form.handleSubmit().then(() => {
              if (!form.state.isValid) focusFirstInvalidField();
              return undefined;
            });
          }}
        >
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
          </DialogHeader>
          <form.Field name="label_id">
            {(field) => (
              <FormField field={field} label="Label" required>
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
              </FormField>
            )}
          </form.Field>
          <FormActions
            form={form}
            submitLabel="Add"
            onCancel={exitGuard.requestDiscard}
            className="justify-end"
          />
        </form>
        {exitGuard.dialog}
      </DialogContent>
    </Dialog>
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
