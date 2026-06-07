import { useForm } from "@tanstack/react-form";
import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Info, Loader2, PackageCheck, Pencil, Plus, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { DataTable, DataTableColumnHeader, DataTableEmptyState } from "@/components/data-table";
import { LabelPicker } from "@/components/labels/label-picker";
import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { FreeTextCombobox } from "@/components/munki/free-text-combobox";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { MutationError } from "@/components/mutation-error";
import { LabelTargetRowsTable } from "@/components/targeting/label-target-rows-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useUploadMunkiArtifact } from "@/hooks/munki/artifacts";
import { type MunkiPackage } from "@/hooks/munki/packages";
import {
  useMunkiSoftwareTitle,
  useMunkiSoftwareTitles,
  useUpdateMunkiSoftwareTitle,
  type MunkiSoftwareTitleDetail,
  type MunkiSoftwareTitleMutation,
} from "@/hooks/munki/software-titles";
import type { MunkiAssignment } from "@/lib/api";
import { fieldErrors, uniqueOptions } from "@/lib/form-validation";
import { selectedLabelTargetIDs } from "@/lib/label-target-rows";
import {
  emptyMunkiAssignmentForm,
  munkiAssignmentDetailsFormSchema,
  munkiAssignmentFormSchema,
  munkiAssignmentIncludeMutation,
  type MunkiAssignmentFormState,
} from "@/lib/munki-assignment-form";
import { softwareTitleFormFromTitle, softwareTitleSchema } from "@/lib/munki-software-title-form";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative } from "@/lib/utils";

import { MunkiAssignmentFormFields } from "@/components/munki/software-title/assignment-form";
import { Sheet, SheetContent, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import {
  munkiAssignmentActionLabel,
  munkiInstallerTypeLabel,
  munkiPackageSelectionLabel,
  munkiRestartActionLabel,
} from "@/lib/munki-software-title";

interface MunkiAssignmentDraft extends Omit<MunkiAssignment, "id" | "label_id"> {
  id: number;
  label_id: number | null;
  draft: true;
}

interface MunkiAssignmentExisting extends Omit<MunkiAssignment, "label_id"> {
  label_id: number | null;
}

type MunkiAssignmentRow = MunkiAssignmentExisting | MunkiAssignmentDraft;

export function MunkiSoftwareTitleDetailPage() {
  const params = useParams({ strict: false });
  const softwareId = Number(params.softwareId);
  const softwareID = Number.isFinite(softwareId) && softwareId > 0 ? softwareId : null;
  const query = useMunkiSoftwareTitle(softwareID);

  if (softwareID === null) {
    return (
      <PageShell>
        <MutationError title="Failed to Load Software" message="Software route is invalid." />
      </PageShell>
    );
  }

  if (query.error) {
    return (
      <PageShell>
        <MutationError title="Failed to Load Software" message={query.error.message} />
      </PageShell>
    );
  }

  if (!query.data) {
    return (
      <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
        <Loader2 className="animate-spin" /> Loading Software...
      </PageShell>
    );
  }

  return (
    <MunkiSoftwareTitleDetailForm
      key={`${query.data.id}:${query.data.updated_at}`}
      software={query.data}
      refetchSoftware={() => query.refetch()}
    />
  );
}

function MunkiSoftwareTitleDetailForm({
  software,
  refetchSoftware,
}: {
  software: MunkiSoftwareTitleDetail;
  refetchSoftware: () => Promise<unknown>;
}) {
  // Category/developer suggestions are loose helper text; MAX_PAGE_SIZE is enough for this non-managed vocabulary.
  const titles = useMunkiSoftwareTitles({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const updateSoftware = useUpdateMunkiSoftwareTitle();
  const iconUpload = useUploadMunkiArtifact("icon");
  const [targetRows, setTargetRows] = useState<MunkiAssignmentRow[]>(() => software.includes ?? []);
  const [nextDraftID, setNextDraftID] = useState(-1);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const [assignmentSheetOpen, setAssignmentSheetOpen] = useState(false);
  const [editingAssignment, setEditingAssignment] = useState<MunkiAssignmentRow | null>(null);
  const [assignmentForm, setAssignmentForm] = useState(() => emptyMunkiAssignmentForm());
  const [showAssignmentErrors, setShowAssignmentErrors] = useState(false);
  const [targetErrors, setTargetErrors] = useState<Partial<Record<number, string>>>({});
  const [excludeForm, setExcludeForm] = useState<number[]>(() => software.exclude_label_ids ?? []);
  const packages = software.packages ?? [];
  const includes = useMemo(() => software.includes ?? [], [software.includes]);
  const excludeLabelIDs = useMemo(() => software.exclude_label_ids ?? [], [software.exclude_label_ids]);
  const includeLabelIDs = useMemo(() => selectedLabelTargetIDs(targetRows), [targetRows]);
  const softwareOptionsForm = useForm({
    defaultValues: softwareTitleFormFromTitle(software),
    validators: {
      onSubmit: softwareTitleSchema,
    },
  });
  const assignmentParsed = useMemo(() => munkiAssignmentDetailsFormSchema.safeParse(assignmentForm), [assignmentForm]);
  const assignmentErrors = useMemo(() => fieldErrors(assignmentParsed), [assignmentParsed]);
  const assignmentMutationError = updateSoftware.error?.message;
  const targetDirty = !sameAssignmentRows(targetRows, includes);
  const excludeDirty = !sameNumberSet(excludeForm, excludeLabelIDs);
  const pagePending = updateSoftware.isPending || iconUpload.isUploading;
  const pageMutationError = updateSoftware.error?.message ?? iconUpload.error?.message;
  const categoryOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.category)),
    [titles.data?.items],
  );
  const developerOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.developer)),
    [titles.data?.items],
  );
  const title = software.name || "Software";

  useEffect(() => {
    setTargetRows(includes);
    setTargetErrors({});
  }, [includes]);

  useEffect(() => {
    setExcludeForm(excludeLabelIDs);
  }, [excludeLabelIDs]);

  function moveAssignments(next: MunkiAssignmentRow[]) {
    updateSoftware.reset();
    setTargetRows(numberAssignmentRows(next));
  }

  function openNewAssignment() {
    updateSoftware.reset();
    setTargetRows((current) =>
      numberAssignmentRows([...current, newDraftAssignment(software, current.length + 1, nextDraftID)]),
    );
    setNextDraftID((current) => current - 1);
  }

  function openEditAssignment(assignment: MunkiAssignmentRow) {
    updateSoftware.reset();
    setEditingAssignment(assignment);
    setAssignmentForm(munkiAssignmentFormFromRow(assignment));
    setShowAssignmentErrors(false);
    setAssignmentSheetOpen(true);
  }

  function saveAssignment() {
    if (!editingAssignment) return;
    const next = munkiAssignmentDetailsFormSchema.safeParse(assignmentForm);
    if (!next.success) {
      setShowAssignmentErrors(true);
      return;
    }

    setTargetRows((current) =>
      current.map((row) =>
        row.id === editingAssignment.id
          ? {
              ...row,
              package_selection: next.data.package_selection,
              pinned_package_id:
                next.data.package_selection === "specific_package" ? Number(next.data.pinned_package_id) : undefined,
              action: next.data.action,
              optional_install: next.data.optional_install,
              featured_item: next.data.featured_item,
            }
          : row,
      ),
    );
    setAssignmentSheetOpen(false);
  }

  function updateAssignmentLabel(id: number, labelID: number | null) {
    updateSoftware.reset();
    setTargetErrors((current) => omitTargetError(current, id));
    setTargetRows((current) => current.map((row) => updateRowLabel(row, id, labelID)));
  }

  function deleteAssignmentRow(assignment: MunkiAssignmentRow) {
    updateSoftware.reset();
    setTargetRows((current) => numberAssignmentRows(current.filter((row) => row.id !== assignment.id)));
    setTargetErrors((current) => omitTargetError(current, assignment.id));
    if (editingAssignment?.id === assignment.id) {
      setAssignmentSheetOpen(false);
      setEditingAssignment(null);
    }
  }

  async function savePage() {
    const softwareDirty = iconFile !== null || iconCleared || softwareOptionsForm.state.isDirty;
    const pageDirty = softwareDirty || targetDirty || excludeDirty;
    if (!pageDirty) return;

    await softwareOptionsForm.handleSubmit();
    const softwareParsed = softwareTitleSchema.safeParse(softwareOptionsForm.state.values);
    if (!softwareParsed.success) return;

    const nextTargets = targetIncludeMutations(targetRows);
    if (!nextTargets.success) {
      setTargetErrors(nextTargets.labelErrors);
      if (nextTargets.editRow) {
        setEditingAssignment(nextTargets.editRow);
        setAssignmentForm(munkiAssignmentFormFromRow(nextTargets.editRow));
        setShowAssignmentErrors(true);
        setAssignmentSheetOpen(true);
      }
      return;
    }

    const iconArtifact = iconFile ? await iconUpload.upload(iconFile) : null;
    const body: MunkiSoftwareTitleMutation = {
      ...softwareParsed.data,
      icon_artifact_id: iconArtifact?.id ?? (iconCleared ? undefined : software.icon_artifact_id),
      includes: nextTargets.includes,
      exclude_label_ids: excludeForm,
    };
    await updateSoftware.mutateAsync({ id: software.id, body });
    setIconFile(null);
    setIconCleared(false);
    await refetchSoftware();
  }

  function resetAssignmentPage() {
    updateSoftware.reset();
    iconUpload.reset();
    softwareOptionsForm.reset(softwareTitleFormFromTitle(software));
    setIconFile(null);
    setIconCleared(false);
    setTargetRows(includes);
    setTargetErrors({});
    setExcludeForm(excludeLabelIDs);
  }

  const packageColumns: ColumnDef<MunkiPackage>[] = [
    {
      id: "version",
      accessorKey: "version",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Version" />,
      cell: ({ row }) => (
        <div className="flex min-w-0 items-center gap-3">
          <PackageIconView pkg={row.original} />
          <div className="min-w-0">
            <div className="truncate font-medium">{row.original.version}</div>
            <div className="text-muted-foreground truncate text-xs">{row.original.software_name}</div>
          </div>
        </div>
      ),
    },
    {
      id: "installer_type",
      header: "Installer",
      enableSorting: false,
      cell: ({ row }) => munkiInstallerTypeLabel(row.original.installer_type),
    },
    {
      id: "behavior",
      header: "Behavior",
      enableSorting: false,
      cell: ({ row }) => <PackageBehavior pkg={row.original} />,
    },
    {
      id: "eligible",
      accessorKey: "eligible",
      header: "Available",
      enableSorting: false,
      cell: ({ row }) => (row.original.eligible ? "Yes" : "No"),
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: "Updated",
      enableSorting: false,
      cell: ({ row }) => formatRelative(row.original.updated_at),
    },
  ];

  const targetDetailColumns: ColumnDef<MunkiAssignmentRow>[] = [
    {
      id: "selection",
      accessorKey: "package_selection",
      header: "Package",
      enableSorting: false,
      cell: ({ row }) => assignmentPackageLabel(row.original),
    },
    {
      id: "action",
      accessorKey: "action",
      header: "Managed",
      enableSorting: false,
      cell: ({ row }) => assignmentActionLabel(row.original),
    },
    {
      id: "msc",
      header: "Availability",
      enableSorting: false,
      cell: ({ row }) => assignmentMSCSections(row.original).join(", ") || "None",
    },
  ];

  return (
    <PageShell>
      <PageHeader title={title} />

      <form
        className="flex flex-col gap-5"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void savePage();
        }}
      >
        <MutableResourceTabs
          tabs={[
            {
              value: "options",
              label: "Options",
              content: (
                <FieldGroup className="max-w-3xl">
                  <div className="flex items-start gap-4">
                    <EditableMunkiIcon
                      title="software icon"
                      iconUrl={iconCleared ? undefined : software.icon_url}
                      file={iconFile}
                      clearable={!!iconFile || (!iconCleared && !!software.icon_artifact_id)}
                      onFileChange={(file) => {
                        setIconFile(file);
                        setIconCleared(false);
                      }}
                      onClear={() => {
                        setIconFile(null);
                        setIconCleared(!!software.icon_artifact_id);
                      }}
                    />
                    <div className="min-w-0 flex-1">
                      <softwareOptionsForm.Field
                        name="name"
                        children={(field) => {
                          const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                          return (
                            <Field data-invalid={invalid}>
                              <FieldLabel htmlFor="munki-software-name" required>
                                Name
                              </FieldLabel>
                              <Input
                                id="munki-software-name"
                                name={field.name}
                                value={field.state.value}
                                aria-invalid={invalid}
                                onBlur={field.handleBlur}
                                onChange={(event) => field.handleChange(event.target.value)}
                              />
                              {invalid ? <FieldError errors={field.state.meta.errors} /> : null}
                            </Field>
                          );
                        }}
                      />
                    </div>
                  </div>
                  <softwareOptionsForm.Field
                    name="description"
                    children={(field) => {
                      const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                      return (
                        <Field data-invalid={invalid}>
                          <FieldLabel htmlFor="munki-software-description">
                            Description
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button
                                  type="button"
                                  className="text-muted-foreground hover:text-foreground inline-flex"
                                  aria-label="Description help"
                                >
                                  <Info className="size-3.5" />
                                </button>
                              </TooltipTrigger>
                              <TooltipContent>Description is shown in Managed Software Center.</TooltipContent>
                            </Tooltip>
                          </FieldLabel>
                          <Textarea
                            id="munki-software-description"
                            name={field.name}
                            value={field.state.value}
                            aria-invalid={invalid}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                          {invalid ? <FieldError errors={field.state.meta.errors} /> : null}
                        </Field>
                      );
                    }}
                  />
                  <div className="grid gap-4 md:grid-cols-2">
                    <softwareOptionsForm.Field
                      name="category"
                      children={(field) => {
                        const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                        return (
                          <Field data-invalid={invalid}>
                            <FieldLabel htmlFor="munki-software-category">Category</FieldLabel>
                            <FreeTextCombobox
                              id="munki-software-category"
                              name={field.name}
                              value={field.state.value}
                              options={categoryOptions}
                              invalid={invalid}
                              onBlur={field.handleBlur}
                              onChange={field.handleChange}
                            />
                            {invalid ? <FieldError errors={field.state.meta.errors} /> : null}
                          </Field>
                        );
                      }}
                    />
                    <softwareOptionsForm.Field
                      name="developer"
                      children={(field) => {
                        const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                        return (
                          <Field data-invalid={invalid}>
                            <FieldLabel htmlFor="munki-software-developer">Developer</FieldLabel>
                            <FreeTextCombobox
                              id="munki-software-developer"
                              name={field.name}
                              value={field.state.value}
                              options={developerOptions}
                              invalid={invalid}
                              onBlur={field.handleBlur}
                              onChange={field.handleChange}
                            />
                            {invalid ? <FieldError errors={field.state.meta.errors} /> : null}
                          </Field>
                        );
                      }}
                    />
                  </div>
                </FieldGroup>
              ),
            },
            {
              value: "scope",
              label: "Scope",
              content: (
                <div className="flex flex-col gap-6">
                  <section className="flex flex-col gap-3">
                    <div className="flex items-center justify-between gap-3">
                      <h2 className="text-base font-semibold">Targets</h2>
                      <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
                        <Button type="button" size="sm" disabled={pagePending} onClick={openNewAssignment}>
                          <Plus data-icon="inline-start" />
                          Add Target
                        </Button>
                      </div>
                    </div>
                    <MutationError title="Failed to Save Target" message={assignmentMutationError} />
                    <LabelTargetRowsTable
                      rows={targetRows}
                      excludeLabelIDs={excludeForm}
                      labelErrors={targetErrors}
                      columnsAfterLabel={targetDetailColumns}
                      onChange={moveAssignments}
                      onLabelChange={updateAssignmentLabel}
                      renderActions={(row) => (
                        <div className="flex justify-end gap-1">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon-sm"
                                aria-label="Edit target"
                                onClick={() => openEditAssignment(row)}
                              >
                                <Pencil />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>Edit Target</TooltipContent>
                          </Tooltip>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon-sm"
                                aria-label="Delete target"
                                onClick={() => deleteAssignmentRow(row)}
                              >
                                <Trash2 />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>Delete Target</TooltipContent>
                          </Tooltip>
                        </div>
                      )}
                      empty={<CompactMunkiEmptyState title="No Targets" />}
                      emptyClassName="min-h-32 py-4"
                    />
                  </section>
                  <section className="flex max-w-3xl flex-col gap-3">
                    <h2 className="text-base font-semibold">Exclusions</h2>
                    <LabelPicker
                      value={excludeForm}
                      includeBuiltins={false}
                      unavailableLabelIDs={includeLabelIDs}
                      placeholder="Add Exclusion"
                      emptyMessage="No Labels Found."
                      onChange={(next) => {
                        updateSoftware.reset();
                        setExcludeForm(next);
                      }}
                    />
                  </section>
                </div>
              ),
            },
            {
              value: "packages",
              label: "Packages",
              content: (
                <div className="flex flex-col gap-3">
                  <div className="flex items-center justify-between gap-3">
                    <h2 className="text-base font-semibold">Packages</h2>
                    <Button asChild size="sm" variant="outline">
                      <Link
                        to="/munki/software-titles/$softwareId/packages/new"
                        params={{ softwareId: String(software.id) }}
                      >
                        <Plus data-icon="inline-start" />
                        Add Package
                      </Link>
                    </Button>
                  </div>
                  <DataTable
                    columns={packageColumns}
                    data={packages}
                    totalCount={packages.length}
                    pagination={{ pageIndex: 0, pageSize: Math.max(packages.length, 1) }}
                    sorting={[]}
                    onPaginationChange={() => undefined}
                    onSortingChange={() => undefined}
                    isLoading={false}
                    clientSort
                    rowHref={(row) => ({
                      to: "/munki/software-titles/$softwareId/packages/$packageId/edit",
                      params: { softwareId: String(row.software_id), packageId: String(row.id) },
                    })}
                    empty={<CompactMunkiEmptyState title="No Packages" />}
                    emptyClassName="min-h-32 py-4"
                  />
                </div>
              ),
            },
          ]}
        />

        <MutationError title="Failed to Save Software" message={pageMutationError} />
        <softwareOptionsForm.Subscribe selector={(state) => state.isDirty}>
          {(softwareFormDirty) => {
            const softwareDirty = iconFile !== null || iconCleared || softwareFormDirty;
            const pageDirty = softwareDirty || excludeDirty || targetDirty;

            return (
              <div className="flex items-center gap-2 border-t pt-4">
                <Button type="submit" size="sm" disabled={!pageDirty || pagePending}>
                  {pagePending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
                  Save
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={!pageDirty || pagePending}
                  onClick={resetAssignmentPage}
                >
                  Cancel
                </Button>
              </div>
            );
          }}
        </softwareOptionsForm.Subscribe>
      </form>
      <Sheet open={assignmentSheetOpen} onOpenChange={setAssignmentSheetOpen}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-2xl">
          <form
            className="flex min-h-0 flex-1 flex-col"
            noValidate
            onSubmit={(event) => {
              event.preventDefault();
              void saveAssignment();
            }}
          >
            <SheetHeader>
              <SheetTitle>Edit Target</SheetTitle>
            </SheetHeader>
            <div className="flex-1 overflow-y-auto px-4 pb-4">
              <MutationError title="Failed to Save Include" message={assignmentMutationError} />
              <MunkiAssignmentFormFields
                form={assignmentForm}
                packages={packages}
                showErrors={showAssignmentErrors}
                errors={assignmentErrors}
                loadingPackages={false}
                className="max-w-none"
                onChange={setAssignmentForm}
              />
            </div>
            <SheetFooter className="border-t">
              <Button type="submit" size="sm">
                Save
              </Button>
              <Button type="button" variant="outline" size="sm" onClick={() => setAssignmentSheetOpen(false)}>
                Cancel
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>
    </PageShell>
  );
}

function newDraftAssignment(software: MunkiSoftwareTitleDetail, priority: number, id: number): MunkiAssignmentDraft {
  const now = new Date().toISOString();
  return {
    id,
    draft: true,
    software_id: software.id,
    software_name: software.name,
    priority,
    label_id: null,
    action: "install",
    optional_install: false,
    featured_item: false,
    package_selection: "latest_eligible",
    created_at: now,
    updated_at: now,
  };
}

function numberAssignmentRows(rows: MunkiAssignmentRow[]) {
  return rows.map((row, index) => ({ ...row, priority: index + 1 }));
}

function munkiAssignmentFormFromRow(row: MunkiAssignmentRow): MunkiAssignmentFormState {
  return {
    priority: row.priority,
    label_id: row.label_id,
    package_selection: row.package_selection,
    pinned_package_id: row.pinned_package_id ? String(row.pinned_package_id) : "",
    action: row.action,
    optional_install: row.optional_install,
    featured_item: row.featured_item,
  };
}

function targetIncludeMutations(rows: MunkiAssignmentRow[]) {
  const includes: ReturnType<typeof munkiAssignmentIncludeMutation>[] = [];
  const labelErrors: Partial<Record<number, string>> = {};
  let editRow: MunkiAssignmentRow | null = null;
  for (const [index, row] of rows.entries()) {
    const form = { ...munkiAssignmentFormFromRow(row), priority: index + 1 };
    const parsed = munkiAssignmentFormSchema.safeParse(form);
    if (!parsed.success) {
      const rowErrors = fieldErrors(parsed);
      if (rowErrors.label_id) {
        labelErrors[row.id] = rowErrors.label_id;
      } else {
        editRow ??= row;
      }
      continue;
    }
    includes.push(munkiAssignmentIncludeMutation(parsed.data));
  }
  if (Object.keys(labelErrors).length > 0 || editRow) {
    return { success: false as const, labelErrors, editRow };
  }
  return { success: true as const, includes };
}

function omitTargetError(errors: Partial<Record<number, string>>, id: number) {
  const next = { ...errors };
  delete next[id];
  return next;
}

function updateRowLabel(row: MunkiAssignmentRow, id: number, labelID: number | null): MunkiAssignmentRow {
  if (row.id !== id) return row;
  if ("draft" in row) return { ...row, draft: true, label_id: labelID };
  return { ...row, label_id: labelID };
}

function sameAssignmentRows(a: MunkiAssignmentRow[], b: MunkiAssignment[]) {
  if (a.length !== b.length) return false;
  return a.every((row, index) => sameAssignmentRow(row, b[index], index + 1));
}

function sameAssignmentRow(a: MunkiAssignmentRow, b: MunkiAssignment | undefined, priority: number) {
  if (!b) return false;
  return (
    a.label_id === b.label_id &&
    priority === b.priority &&
    a.action === b.action &&
    a.optional_install === b.optional_install &&
    a.featured_item === b.featured_item &&
    a.package_selection === b.package_selection &&
    (a.pinned_package_id ?? null) === (b.pinned_package_id ?? null)
  );
}

function assignmentPackageLabel(assignment: MunkiAssignmentRow) {
  if (assignment.package_selection === "specific_package") {
    return assignment.pinned_package_version
      ? `${assignment.pinned_package_name ?? "Pinned"} ${assignment.pinned_package_version}`
      : munkiPackageSelectionLabel(assignment.package_selection);
  }
  return munkiPackageSelectionLabel(assignment.package_selection);
}

function assignmentActionLabel(assignment: MunkiAssignmentRow) {
  return munkiAssignmentActionLabel(assignment.action);
}

function assignmentMSCSections(assignment: MunkiAssignmentRow) {
  const sections: string[] = [];
  if (assignment.optional_install) sections.push("Optional Installs");
  if (assignment.featured_item) sections.push("Featured Items");
  return sections;
}

function sameNumberSet(a: readonly number[], b: readonly number[]) {
  if (a.length !== b.length) return false;
  const values = new Set(a);
  if (values.size !== b.length) return false;
  return b.every((value) => values.has(value));
}

function CompactMunkiEmptyState({ title }: { title: string }) {
  return <DataTableEmptyState icon={<PackageCheck />} title={title} className="gap-3 p-4 md:p-4" />;
}

function PackageIconView({ pkg }: { pkg: MunkiPackage }) {
  return <MunkiIcon iconUrl={pkg.icon_url} size="md" />;
}

function PackageBehavior({ pkg }: { pkg: MunkiPackage }) {
  const values = [
    pkg.unattended_install ? "Unattended" : "",
    pkg.uninstall_method !== "none" ? "Uninstallable" : "",
    pkg.on_demand ? "On demand" : "",
    pkg.restart_action && pkg.restart_action !== "None" ? munkiRestartActionLabel(pkg.restart_action) : "",
  ].filter(Boolean);
  if (values.length === 0) {
    return <span className="text-muted-foreground">Standard</span>;
  }
  return (
    <div className="flex flex-wrap gap-1">
      {values.map((value) => (
        <Badge key={value} variant="secondary" className="font-normal">
          {value}
        </Badge>
      ))}
    </div>
  );
}
