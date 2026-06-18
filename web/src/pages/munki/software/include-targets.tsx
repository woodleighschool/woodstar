import { encodeSort } from "@/hooks/use-data-table-search";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { GripVertical, MoreHorizontal, Plus } from "lucide-react";
import { useMemo, useState } from "react";

import { EmptyPanel } from "@/components/empty-panel";
import { AssignmentLabelField } from "@/components/targeting/assignment-label-field";
import { TargetSection } from "@/components/targeting/target-section";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxAnchor,
  ComboboxBadgeItem,
  ComboboxBadgeList,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxTrigger,
} from "@/components/ui/combobox";
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
import { Field, FieldLabel } from "@/components/ui/field";
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
import { useLabels } from "@/hooks/use-labels";
import { type MunkiPackage } from "@/hooks/use-munki-packages";
import type { MunkiInclude } from "@/lib/api";

import {
  LATEST_PACKAGE_VALUE,
  munkiSoftwareTargetSchema,
  targetPackageFromValue,
  targetPackageValue,
} from "./fields";
import { MUNKI_SOFTWARE_ACTION_OPTIONS } from "./munki-software";

export interface MunkiSoftwareTargetRow {
  id: number;
  label_id: number | null;
  priority: number;
  package: MunkiInclude["package"];
  actions: MunkiInclude["actions"];
}

interface TargetDraft {
  label_id: number | null;
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
  const [draft, setDraft] = useState<TargetDraft>(emptyDraft);
  const labels = useLabels({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const labelsByID = useMemo(
    () =>
      new Map<number, string>((labels.data?.items ?? []).map((label) => [label.id, label.name])),
    [labels.data],
  );
  const usedLabelIDs = rows.flatMap((row) => (row.label_id === null ? [] : [row.label_id]));
  const unavailableLabelIDs = [...usedLabelIDs, ...excludeLabelIDs];
  const canSave = munkiSoftwareTargetSchema.safeParse({ ...draft, priority: 1 }).success;

  function openAdd() {
    setDraft(emptyDraft());
    setDialog({ mode: "add" });
  }

  function openEdit(row: MunkiSoftwareTargetRow) {
    setDraft({ label_id: row.label_id, package: row.package, actions: row.actions });
    setDialog({ mode: "edit", id: row.id });
  }

  function save() {
    if (!canSave || dialog === null) return;
    if (dialog.mode === "add") {
      onChange([
        ...rows,
        {
          id: nextTargetID(rows),
          priority: rows.length + 1,
          label_id: draft.label_id,
          package: draft.package,
          actions: draft.actions,
        },
      ]);
    } else {
      onChange(
        rows.map((row) =>
          row.id === dialog.id ? { ...row, package: draft.package, actions: draft.actions } : row,
        ),
      );
    }
    setDialog(null);
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

      <Dialog open={dialog !== null} onOpenChange={(open) => (open ? null : setDialog(null))}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{dialog?.mode === "edit" ? "Edit Include" : "Add Include"}</DialogTitle>
          </DialogHeader>

          {dialog?.mode === "add" ? (
            <AssignmentLabelField
              value={draft.label_id}
              onChange={(label_id) => setDraft((current) => ({ ...current, label_id }))}
              unavailableLabelIDs={unavailableLabelIDs}
            />
          ) : null}

          <Field>
            <FieldLabel htmlFor="munki-target-package">Package</FieldLabel>
            <Select
              value={targetPackageValue(draft.package)}
              onValueChange={(value) =>
                setDraft((current) => ({ ...current, package: targetPackageFromValue(value) }))
              }
            >
              <SelectTrigger id="munki-target-package" className="w-full">
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
          </Field>

          <TargetActionsField
            value={draft.actions}
            onChange={(actions) => setDraft((current) => ({ ...current, actions }))}
          />

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setDialog(null)}>
              Cancel
            </Button>
            <Button type="button" onClick={save} disabled={!canSave}>
              {dialog?.mode === "edit" ? "Save" : "Add"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </TargetSection>
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

function TargetActionsField({
  value,
  onChange,
}: {
  value: MunkiInclude["actions"];
  onChange: (actions: MunkiInclude["actions"]) => void;
}) {
  const selected = MUNKI_SOFTWARE_ACTION_OPTIONS.filter((option) => value.includes(option.value));
  const warning = targetActionWarning(value);

  return (
    <Field>
      <FieldLabel>Actions</FieldLabel>
      <Combobox
        multiple
        value={value}
        onValueChange={(next) =>
          onChange(
            next.filter((action): action is MunkiInclude["actions"][number] =>
              MUNKI_SOFTWARE_ACTION_OPTIONS.some((option) => option.value === action),
            ),
          )
        }
      >
        <ComboboxAnchor className="h-auto min-h-9 flex-wrap py-1.5 pr-2">
          <ComboboxBadgeList>
            {selected.map((option) => (
              <ComboboxBadgeItem key={option.value} value={option.value}>
                {option.label}
              </ComboboxBadgeItem>
            ))}
          </ComboboxBadgeList>
          <ComboboxInput
            className="h-[calc(--spacing(5.5))] min-w-16 flex-1 px-0 py-0 text-sm"
            placeholder={selected.length === 0 ? "Pick actions" : ""}
            required={selected.length === 0}
          />
          <ComboboxTrigger aria-label="Open actions" className="ml-auto" />
        </ComboboxAnchor>
        <ComboboxContent>
          <ComboboxEmpty>No Actions Found.</ComboboxEmpty>
          {MUNKI_SOFTWARE_ACTION_OPTIONS.map((option) => (
            <ComboboxItem key={option.value} value={option.value} label={option.label}>
              <span className="min-w-0 flex-1 truncate">{option.label}</span>
            </ComboboxItem>
          ))}
        </ComboboxContent>
      </Combobox>
      {warning ? <p className="text-xs text-warning">{warning}</p> : null}
    </Field>
  );
}

function emptyDraft(): TargetDraft {
  return { label_id: null, package: { strategy: "latest" }, actions: [] };
}

function packageLabel(pkg: MunkiInclude["package"], packages: MunkiPackage[]) {
  if (pkg.strategy === "specific") {
    return (
      packages.find((item) => item.id === pkg.package_id)?.version ?? `Package ${pkg.package_id}`
    );
  }
  return "Latest";
}

function targetActionWarning(actions: MunkiInclude["actions"]) {
  if (actions.includes("featured_items") && !actions.includes("optional_installs")) {
    return "Munki ignores featured items unless the item is also optional.";
  }
  return "";
}

function nextTargetID(rows: MunkiSoftwareTargetRow[]) {
  return rows.reduce((max, row) => Math.max(max, row.id), 0) + 1;
}
