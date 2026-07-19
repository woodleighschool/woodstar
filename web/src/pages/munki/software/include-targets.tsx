import { revalidateLogic, useForm } from "@tanstack/react-form";
import { GripVertical, MoreHorizontal, Plus } from "lucide-react";
import { useMemo, useState } from "react";

import {
  DraggableTableRow,
  DraggableTableRowHandle,
  DraggableTableRows,
} from "@/components/data-table/draggable-table-rows";
import { EmptyPanel } from "@/components/empty-panel";
import { FormActions } from "@/components/form-actions";
import { FormField } from "@/components/form-field";
import { focusFirstInvalidField } from "@/components/form-tabs";
import { LabelPicker } from "@/components/labels/label-picker";
import { TargetSection } from "@/components/targeting/target-section";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { encodeSort } from "@/hooks/use-data-table-search";
import { useLabels } from "@/hooks/use-labels";
import type { MunkiInclude, MunkiPackage } from "@/lib/api";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import {
  LATEST_PACKAGE_VALUE,
  munkiSoftwareTargetSchema,
  targetPackageFromValue,
  targetPackageValue,
} from "./fields";
import {
  MUNKI_ASSIGNMENT_ACTION_VALUES,
  MUNKI_OPTIONAL_PRESENTATION_ACTION_VALUES,
  MUNKI_SOFTWARE_ACTION_OPTIONS,
  MUNKI_SOFTWARE_ACTION_VALUES,
  MUNKI_SOFTWARE_ACTIONS,
} from "./munki-software";
export interface MunkiSoftwareTargetRow {
  id: number;
  label_id: number | null;
  priority: number;
  package: MunkiInclude["package"];
  actions: MunkiInclude["actions"];
}
type DialogState =
  | {
      mode: "add";
    }
  | {
      mode: "edit";
      id: number;
    }
  | null;
export function MunkiIncludeTargets({
  rows,
  excludeLabelIDs,
  packages,
  onChange,
}: {
  rows: MunkiSoftwareTargetRow[];
  excludeLabelIDs: number[];
  packages: MunkiPackage[];
  onChange: (rows: MunkiSoftwareTargetRow[]) => void;
}) {
  const [dialog, setDialog] = useState<DialogState>(null);
  const labels = useLabels({
    per_page: MAX_PAGE_SIZE,
    sort: encodeSort("name"),
  });
  const labelsByID = useMemo(
    () =>
      new Map<number, string>((labels.data?.items ?? []).map((label) => [label.id, label.name])),
    [labels.data],
  );
  const usedLabelIDs = rows.flatMap((row) => (row.label_id === null ? [] : [row.label_id]));
  const unavailableLabelIDs = [...usedLabelIDs, ...excludeLabelIDs];
  function openAdd() {
    setDialog({ mode: "add" });
  }
  function openEdit(row: MunkiSoftwareTargetRow) {
    setDialog({ mode: "edit", id: row.id });
  }
  return (
    <TargetSection
      title="Include"
      action={
        <Button type="button" variant="outline" size="sm" onClick={openAdd}>
          <Plus data-icon="inline-start" />
          Add Include
        </Button>
      }
    >
      {rows.length > 0 ? (
        <DraggableTableRows
          value={rows}
          onValueChange={(next) =>
            onChange(next.map((row, index) => ({ ...row, priority: index + 1 })))
          }
          getRowId={(row) => row.id}
        >
          <div className="overflow-x-auto rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-10" />
                  <TableHead>Label</TableHead>
                  <TableHead>Package</TableHead>
                  <TableHead>Actions</TableHead>
                  <TableHead className="w-12" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((row) => (
                  <DraggableTableRow key={row.id} id={row.id}>
                    <TableCell className="w-10">
                      <DraggableTableRowHandle disabled={rows.length <= 1}>
                        <GripVertical className="text-muted-foreground" />
                      </DraggableTableRowHandle>
                    </TableCell>
                    <TableCell>
                      {row.label_id === null
                        ? "-"
                        : (labelsByID.get(row.label_id) ?? `Label ${row.label_id}`)}
                    </TableCell>
                    <TableCell>{packageLabel(row.package, packages)}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {MUNKI_SOFTWARE_ACTION_OPTIONS.filter((option) =>
                          row.actions.includes(option.value),
                        ).map((option) => (
                          <Badge key={option.value} variant="secondary" className="font-normal">
                            {option.label}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="w-12">
                      <MunkiIncludeRowActions
                        onEdit={() => openEdit(row)}
                        onRemove={() => onChange(rows.filter((item) => item.id !== row.id))}
                      />
                    </TableCell>
                  </DraggableTableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </DraggableTableRows>
      ) : (
        <EmptyPanel>No includes yet</EmptyPanel>
      )}

      {dialog ? (
        <MunkiIncludeDialog
          mode={dialog.mode}
          initial={
            dialog.mode === "edit"
              ? requiredTarget(rows, dialog.id)
              : {
                  id: nextTargetID(rows),
                  priority: rows.length + 1,
                  label_id: null,
                  package: { strategy: "latest" },
                  actions: [],
                }
          }
          packages={packages}
          unavailableLabelIDs={unavailableLabelIDs}
          onClose={() => setDialog(null)}
          onSave={(target) => {
            onChange(
              dialog.mode === "add"
                ? [...rows, target]
                : rows.map((row) => (row.id === target.id ? target : row)),
            );
            setDialog(null);
          }}
        />
      ) : null}
    </TargetSection>
  );
}
function MunkiIncludeDialog({
  mode,
  initial,
  packages,
  unavailableLabelIDs,
  onClose,
  onSave,
}: {
  mode: "add" | "edit";
  initial: MunkiSoftwareTargetRow;
  packages: MunkiPackage[];
  unavailableLabelIDs: readonly number[];
  onClose: () => void;
  onSave: (target: MunkiSoftwareTargetRow) => void;
}) {
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({
      mode: "submit",
      modeAfterSubmission: "change",
    }),
    validators: { onDynamic: munkiSoftwareTargetSchema },
    onSubmit: ({ value }) => onSave(munkiSoftwareTargetSchema.parse(value)),
  });
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent className="sm:max-w-lg">
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
            <DialogTitle>{mode === "edit" ? "Edit Include" : "Add Include"}</DialogTitle>
          </DialogHeader>

          {mode === "add" ? (
            <form.Field name="label_id">
              {(field) => (
                <FormField field={field} label="Label" required>
                  {(control) => (
                    <LabelPicker
                      value={field.state.value === null ? [] : [field.state.value]}
                      onChange={(ids) => field.handleChange(ids[0] ?? null)}
                      selectionMode="single"
                      includeBuiltins
                      unavailableLabelIDs={unavailableLabelIDs}
                      required
                      invalid={control["aria-invalid"]}
                      placeholder="Select Label"
                    />
                  )}
                </FormField>
              )}
            </form.Field>
          ) : null}

          <form.Field name="package">
            {(field) => (
              <FormField field={field} label="Package" htmlFor="munki-target-package" required>
                {(control) => (
                  <Select
                    items={[
                      { value: LATEST_PACKAGE_VALUE, label: "Latest" },
                      ...packages.map((pkg) => ({ value: String(pkg.id), label: pkg.version })),
                    ]}
                    value={targetPackageValue(field.state.value)}
                    onValueChange={(value) =>
                      field.handleChange(targetPackageFromValue(value ?? LATEST_PACKAGE_VALUE))
                    }
                  >
                    <SelectTrigger {...control} className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem value={LATEST_PACKAGE_VALUE}>Latest</SelectItem>
                        {packages.length > 0 ? <SelectSeparator /> : null}
                        {packages.map((pkg) => (
                          <SelectItem key={pkg.id} value={String(pkg.id)}>
                            {pkg.version}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                )}
              </FormField>
            )}
          </form.Field>

          <form.Field name="actions">
            {(field) => (
              <FormField field={field} label="Actions" required>
                {(control) => (
                  <TargetActionsField
                    value={field.state.value}
                    onChange={field.handleChange}
                    invalid={control["aria-invalid"]}
                  />
                )}
              </FormField>
            )}
          </form.Field>

          <FormActions
            form={form}
            submitLabel={mode === "edit" ? "Save" : "Add"}
            onCancel={onClose}
            className="justify-end"
          />
        </form>
      </DialogContent>
    </Dialog>
  );
}
function MunkiIncludeRowActions({
  onEdit,
  onRemove,
}: {
  onEdit: () => void;
  onRemove: () => void;
}) {
  return (
    <div className="flex justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label="Open include actions"
            />
          }
        >
          <MoreHorizontal />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-36">
          <DropdownMenuGroup>
            <DropdownMenuItem onClick={onEdit}>Edit</DropdownMenuItem>
            <DropdownMenuItem variant="destructive" onClick={onRemove}>
              Delete
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
const MUNKI_EXCLUSIVE_ACTION_VALUES = ["managed_installs", "managed_uninstalls"] as const;

function TargetActionsField({
  value,
  onChange,
  invalid,
}: {
  value: MunkiInclude["actions"];
  onChange: (actions: MunkiInclude["actions"]) => void;
  invalid?: boolean;
}) {
  const selectedAssignmentActions = MUNKI_ASSIGNMENT_ACTION_VALUES.filter((action) =>
    value.includes(action),
  );
  const hasExclusiveAction = MUNKI_EXCLUSIVE_ACTION_VALUES.some((action) => value.includes(action));
  const updateAction = (action: MunkiInclude["actions"][number], checked: boolean) => {
    const next = new Set(value);
    if (checked) {
      next.add(action);
    } else {
      next.delete(action);
      if (action === "optional_installs") {
        MUNKI_OPTIONAL_PRESENTATION_ACTION_VALUES.forEach((optionalAction) =>
          next.delete(optionalAction),
        );
      }
    }
    onChange(MUNKI_SOFTWARE_ACTION_VALUES.filter((candidate) => next.has(candidate)));
  };
  return (
    <>
      <FieldSet className="grid gap-3" aria-invalid={invalid ? true : undefined}>
        <FieldLegend className="sr-only">Actions</FieldLegend>
        {MUNKI_ASSIGNMENT_ACTION_VALUES.map((action) => {
          const id = `munki-target-action-${action}`;
          const checked = value.includes(action);
          const isExclusiveAction =
            action === "managed_installs" || action === "managed_uninstalls";
          const disabled =
            !checked &&
            (hasExclusiveAction || (isExclusiveAction && selectedAssignmentActions.length > 0));
          return (
            <Field
              key={action}
              orientation="horizontal"
              className={disabled ? "opacity-60" : undefined}
            >
              <Checkbox
                id={id}
                checked={checked}
                disabled={disabled}
                onCheckedChange={(state) => updateAction(action, state)}
              />
              <FieldContent>
                <FieldLabel htmlFor={id}>{MUNKI_SOFTWARE_ACTIONS[action].name}</FieldLabel>
                <FieldDescription>{MUNKI_SOFTWARE_ACTIONS[action].description}</FieldDescription>
              </FieldContent>
            </Field>
          );
        })}
      </FieldSet>
      {value.includes("optional_installs") ? (
        <div className="grid gap-3 pl-6">
          {MUNKI_OPTIONAL_PRESENTATION_ACTION_VALUES.map((action) => {
            const id = `munki-target-action-${action}`;
            const checked = value.includes(action);
            return (
              <Field key={action} orientation="horizontal">
                <Checkbox
                  id={id}
                  checked={checked}
                  onCheckedChange={(state) => updateAction(action, state)}
                />
                <FieldContent>
                  <FieldLabel htmlFor={id}>{MUNKI_SOFTWARE_ACTIONS[action].name}</FieldLabel>
                  <FieldDescription>{MUNKI_SOFTWARE_ACTIONS[action].description}</FieldDescription>
                </FieldContent>
              </Field>
            );
          })}
        </div>
      ) : null}
    </>
  );
}
function packageLabel(pkg: MunkiInclude["package"], packages: MunkiPackage[]) {
  if (pkg.strategy === "specific") {
    const selected = packages.find((item) => item.id === pkg.package_id);
    if (!selected) throw new Error(`Munki package ${pkg.package_id} is missing from its software`);
    return selected.version;
  }
  return "Latest";
}
function nextTargetID(rows: MunkiSoftwareTargetRow[]) {
  return rows.reduce((max, row) => Math.max(max, row.id), 0) + 1;
}
function requiredTarget(rows: MunkiSoftwareTargetRow[], id: number) {
  const target = rows.find((row) => row.id === id);
  if (!target) throw new Error("The selected Munki include no longer exists");
  return target;
}
