import { revalidateLogic, useForm } from "@tanstack/react-form";
import { GripVertical, MoreHorizontal, Plus } from "lucide-react";
import { useMemo, useState } from "react";

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
import { Field, FieldContent, FieldDescription, FieldLabel } from "@/components/ui/field";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
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
  Sortable,
  SortableContent,
  SortableItem,
  SortableItemHandle,
} from "@/components/ui/sortable";
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
import type { MunkiInclude, MunkiPackage } from "@/lib/api";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import {
  LATEST_PACKAGE_VALUE,
  munkiSoftwareTargetSchema,
  targetPackageFromValue,
  targetPackageValue,
} from "./fields";
import {
  MUNKI_OPTIONAL_MODIFIER_VALUES,
  MUNKI_PRIMARY_ACTION_VALUES,
  MUNKI_SOFTWARE_ACTION_OPTIONS,
  MUNKI_SOFTWARE_ACTIONS,
  type MunkiPrimaryAction,
} from "./munki-software";

export interface MunkiSoftwareTargetRow {
  id: number;
  label_id: number | null;
  priority: number;
  package: MunkiInclude["package"];
  actions: MunkiInclude["actions"];
}

type DialogState = { mode: "add" } | { mode: "edit"; id: number } | null;

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
        <Sortable
          value={rows}
          onValueChange={(next) =>
            onChange(next.map((row, index) => ({ ...row, priority: index + 1 })))
          }
          getItemValue={(row) => row.id}
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
              <SortableContent asChild>
                <TableBody>
                  {rows.map((row) => (
                    <SortableItem key={row.id} value={row.id} asChild>
                      <TableRow>
                        <TableCell className="w-10">
                          <SortableItemHandle asChild disabled={rows.length <= 1}>
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              aria-label="Drag to reorder"
                            >
                              <GripVertical className="text-muted-foreground" />
                            </Button>
                          </SortableItemHandle>
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
                      </TableRow>
                    </SortableItem>
                  ))}
                </TableBody>
              </SortableContent>
            </Table>
          </div>
        </Sortable>
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
                    value={targetPackageValue(field.state.value)}
                    onValueChange={(value) => field.handleChange(targetPackageFromValue(value))}
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
            onCancel={exitGuard.requestDiscard}
            className="justify-end"
          />
        </form>
        {exitGuard.dialog}
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
        <DropdownMenuTrigger asChild>
          <Button type="button" variant="ghost" size="icon-sm" aria-label="Open include actions">
            <MoreHorizontal />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-36">
          <DropdownMenuGroup>
            <DropdownMenuItem onSelect={onEdit}>Edit</DropdownMenuItem>
            <DropdownMenuItem variant="destructive" onSelect={onRemove}>
              Delete
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

interface ActionSelection {
  intent: MunkiPrimaryAction | "";
  featured: boolean;
  preselect: boolean;
}

function selectionFromActions(actions: MunkiInclude["actions"]): ActionSelection {
  return {
    intent: MUNKI_PRIMARY_ACTION_VALUES.find((action) => actions.includes(action)) ?? "",
    featured: actions.includes("featured_items"),
    preselect: actions.includes("default_installs"),
  };
}

function actionsFromSelection(selection: ActionSelection): MunkiInclude["actions"] {
  if (selection.intent === "") return [];
  if (selection.intent !== "optional_installs") return [selection.intent];
  const actions: MunkiInclude["actions"] = ["optional_installs"];
  if (selection.featured) actions.push("featured_items");
  if (selection.preselect) actions.push("default_installs");
  return actions;
}

function TargetActionsField({
  value,
  onChange,
  invalid,
}: {
  value: MunkiInclude["actions"];
  onChange: (actions: MunkiInclude["actions"]) => void;
  invalid?: boolean;
}) {
  const selection = selectionFromActions(value);
  const update = (next: Partial<ActionSelection>) =>
    onChange(actionsFromSelection({ ...selection, ...next }));

  return (
    <>
      <RadioGroup
        value={selection.intent}
        aria-invalid={invalid ? true : undefined}
        onValueChange={(intent) => update({ intent: intent as MunkiPrimaryAction })}
      >
        {MUNKI_PRIMARY_ACTION_VALUES.map((action) => {
          const id = `munki-target-action-${action}`;
          return (
            <Field key={action} orientation="horizontal">
              <RadioGroupItem id={id} value={action} />
              <FieldContent>
                <FieldLabel htmlFor={id}>{MUNKI_SOFTWARE_ACTIONS[action].name}</FieldLabel>
                <FieldDescription>{MUNKI_SOFTWARE_ACTIONS[action].description}</FieldDescription>
              </FieldContent>
            </Field>
          );
        })}
      </RadioGroup>
      {selection.intent === "optional_installs" ? (
        <div className="grid gap-3 pl-6">
          {MUNKI_OPTIONAL_MODIFIER_VALUES.map((modifier) => {
            const id = `munki-target-action-${modifier}`;
            const checked =
              modifier === "featured_items" ? selection.featured : selection.preselect;
            return (
              <Field key={modifier} orientation="horizontal">
                <Checkbox
                  id={id}
                  checked={checked}
                  onCheckedChange={(state) =>
                    update(
                      modifier === "featured_items"
                        ? { featured: state === true }
                        : { preselect: state === true },
                    )
                  }
                />
                <FieldContent>
                  <FieldLabel htmlFor={id}>{MUNKI_SOFTWARE_ACTIONS[modifier].name}</FieldLabel>
                  <FieldDescription>
                    {MUNKI_SOFTWARE_ACTIONS[modifier].description}
                  </FieldDescription>
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
