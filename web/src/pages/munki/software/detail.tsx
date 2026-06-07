import { useForm } from "@tanstack/react-form";
import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Info, Loader2, PackageCheck, Pencil, Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

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
  useMunkiSoftware,
  useMunkiSoftwareDetail,
  useUpdateMunkiSoftware,
  type MunkiSoftwareDetail,
  type MunkiSoftwareMutation,
} from "@/hooks/munki/software";
import type { SoftwareInclude } from "@/lib/api";
import { fieldErrors, uniqueOptions } from "@/lib/form-validation";
import { selectedLabelTargetIDs } from "@/lib/label-target-rows";
import { munkiSoftwareFormFromSoftware, munkiSoftwareSchema } from "@/lib/munki-software-form";
import {
  emptyMunkiSoftwareTargetForm,
  munkiSoftwareInclude,
  munkiSoftwareTargetDetailsFormSchema,
  munkiSoftwareTargetFormSchema,
  type MunkiSoftwareTargetFormState,
} from "@/lib/munki-software-target-form";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative } from "@/lib/utils";

import { MunkiSoftwareTargetFormFields } from "@/components/munki/software/target-form";
import { Sheet, SheetContent, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import {
  munkiInstallerTypeLabel,
  munkiPackageStrategyLabel,
  munkiRestartActionLabel,
  munkiSoftwareStateLabel,
} from "@/lib/munki-software";

interface MunkiSoftwareTargetRow {
  id: number;
  label_id: number | null;
  priority: number;
  package: SoftwareInclude["package"];
  state: SoftwareInclude["state"];
  featured: boolean;
}

export function MunkiSoftwareDetailPage() {
  const params = useParams({ strict: false });
  const softwareId = Number(params.softwareId);
  const softwareID = Number.isFinite(softwareId) && softwareId > 0 ? softwareId : null;
  const query = useMunkiSoftwareDetail(softwareID);

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
    <MunkiSoftwareDetailForm
      key={`${query.data.id}:${query.data.updated_at}`}
      software={query.data}
      refetchSoftware={() => query.refetch()}
    />
  );
}

function MunkiSoftwareDetailForm({
  software,
  refetchSoftware,
}: {
  software: MunkiSoftwareDetail;
  refetchSoftware: () => Promise<unknown>;
}) {
  // Category/developer suggestions are loose helper text; MAX_PAGE_SIZE is enough for this non-managed vocabulary.
  const titles = useMunkiSoftware({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const updateSoftware = useUpdateMunkiSoftware();
  const iconUpload = useUploadMunkiArtifact("icon");
  const [targetRows, setTargetRows] = useState<MunkiSoftwareTargetRow[]>(() =>
    targetRowsFromIncludes(software.targets.include),
  );
  const [nextDraftID, setNextDraftID] = useState(-1);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const [targetSheetOpen, setTargetSheetOpen] = useState(false);
  const [editingTarget, setEditingTarget] = useState<MunkiSoftwareTargetRow | null>(null);
  const [targetForm, setTargetForm] = useState(() => emptyMunkiSoftwareTargetForm());
  const [showTargetErrors, setShowTargetErrors] = useState(false);
  const [targetErrors, setTargetErrors] = useState<Partial<Record<number, string>>>({});
  const [excludeForm, setExcludeForm] = useState<number[]>(() => excludeLabelIDsFromTargets(software));
  const packages = software.packages ?? [];
  const includes = useMemo(() => software.targets.include, [software.targets.include]);
  const excludeLabelIDs = useMemo(() => excludeLabelIDsFromTargets(software), [software]);
  const includeLabelIDs = useMemo(() => selectedLabelTargetIDs(targetRows), [targetRows]);
  const softwareOptionsForm = useForm({
    defaultValues: munkiSoftwareFormFromSoftware(software),
    validators: {
      onSubmit: munkiSoftwareSchema,
    },
  });
  const targetParsed = useMemo(() => munkiSoftwareTargetDetailsFormSchema.safeParse(targetForm), [targetForm]);
  const targetFormErrors = useMemo(() => fieldErrors(targetParsed), [targetParsed]);
  const targetMutationError = updateSoftware.error?.message;
  const targetDirty = !sameTargetRows(targetRows, includes);
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

  function moveTargets(next: MunkiSoftwareTargetRow[]) {
    updateSoftware.reset();
    setTargetRows(numberTargetRows(next));
  }

  function openNewTarget() {
    updateSoftware.reset();
    setTargetRows((current) => numberTargetRows([...current, newDraftTarget(current.length + 1, nextDraftID)]));
    setNextDraftID((current) => current - 1);
  }

  function openEditTarget(target: MunkiSoftwareTargetRow) {
    updateSoftware.reset();
    setEditingTarget(target);
    setTargetForm(munkiSoftwareTargetFormFromRow(target));
    setShowTargetErrors(false);
    setTargetSheetOpen(true);
  }

  function saveTarget() {
    if (!editingTarget) return;
    const next = munkiSoftwareTargetDetailsFormSchema.safeParse(targetForm);
    if (!next.success) {
      setShowTargetErrors(true);
      return;
    }

    setTargetRows((current) =>
      current.map((row) =>
        row.id === editingTarget.id
          ? {
              ...row,
              package: {
                strategy: next.data.strategy,
                package_id: next.data.strategy === "specific" ? Number(next.data.package_id) : undefined,
              },
              state: next.data.state,
              featured: next.data.featured,
            }
          : row,
      ),
    );
    setTargetSheetOpen(false);
  }

  function updateTargetLabel(id: number, labelID: number | null) {
    updateSoftware.reset();
    setTargetErrors((current) => omitTargetError(current, id));
    setTargetRows((current) => current.map((row) => updateRowLabel(row, id, labelID)));
  }

  function deleteTargetRow(target: MunkiSoftwareTargetRow) {
    updateSoftware.reset();
    setTargetRows((current) => numberTargetRows(current.filter((row) => row.id !== target.id)));
    setTargetErrors((current) => omitTargetError(current, target.id));
    if (editingTarget?.id === target.id) {
      setTargetSheetOpen(false);
      setEditingTarget(null);
    }
  }

  async function savePage() {
    const softwareDirty = iconFile !== null || iconCleared || softwareOptionsForm.state.isDirty;
    const pageDirty = softwareDirty || targetDirty || excludeDirty;
    if (!pageDirty) return;

    await softwareOptionsForm.handleSubmit();
    const softwareParsed = munkiSoftwareSchema.safeParse(softwareOptionsForm.state.values);
    if (!softwareParsed.success) return;

    const nextTargets = targetIncludeMutations(targetRows);
    if (!nextTargets.success) {
      setTargetErrors(nextTargets.labelErrors);
      if (nextTargets.editRow) {
        setEditingTarget(nextTargets.editRow);
        setTargetForm(munkiSoftwareTargetFormFromRow(nextTargets.editRow));
        setShowTargetErrors(true);
        setTargetSheetOpen(true);
      }
      return;
    }

    const iconArtifact = iconFile ? await iconUpload.upload(iconFile) : null;
    const body: MunkiSoftwareMutation = {
      ...softwareParsed.data,
      icon_artifact_id: iconArtifact?.id ?? (iconCleared ? undefined : software.icon_artifact_id),
      targets: {
        include: nextTargets.includes,
        exclude: excludeForm.map((label_id) => ({ label_id })),
      },
    };
    await updateSoftware.mutateAsync({ id: software.id, body });
    setIconFile(null);
    setIconCleared(false);
    await refetchSoftware();
  }

  function resetTargetPage() {
    updateSoftware.reset();
    iconUpload.reset();
    softwareOptionsForm.reset(munkiSoftwareFormFromSoftware(software));
    setIconFile(null);
    setIconCleared(false);
    setTargetRows(targetRowsFromIncludes(includes));
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

  const targetDetailColumns: ColumnDef<MunkiSoftwareTargetRow>[] = [
    {
      id: "selection",
      header: "Package",
      enableSorting: false,
      cell: ({ row }) => targetPackageLabel(row.original, packages),
    },
    {
      id: "state",
      accessorKey: "state",
      header: "Managed",
      enableSorting: false,
      cell: ({ row }) => munkiSoftwareStateLabel(row.original.state),
    },
    {
      id: "msc",
      header: "Availability",
      enableSorting: false,
      cell: ({ row }) => targetAvailabilitySections(row.original).join(", ") || "None",
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
                        <Button type="button" size="sm" disabled={pagePending} onClick={openNewTarget}>
                          <Plus data-icon="inline-start" />
                          Add Target
                        </Button>
                      </div>
                    </div>
                    <MutationError title="Failed to Save Target" message={targetMutationError} />
                    <LabelTargetRowsTable
                      rows={targetRows}
                      excludeLabelIDs={excludeForm}
                      labelErrors={targetErrors}
                      columnsAfterLabel={targetDetailColumns}
                      onChange={moveTargets}
                      onLabelChange={updateTargetLabel}
                      renderActions={(row) => (
                        <div className="flex justify-end gap-1">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon-sm"
                                aria-label="Edit target"
                                onClick={() => openEditTarget(row)}
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
                                onClick={() => deleteTargetRow(row)}
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
                      <Link to="/munki/software/$softwareId/packages/new" params={{ softwareId: String(software.id) }}>
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
                      to: "/munki/software/$softwareId/packages/$packageId/edit",
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
                  onClick={resetTargetPage}
                >
                  Cancel
                </Button>
              </div>
            );
          }}
        </softwareOptionsForm.Subscribe>
      </form>
      <Sheet open={targetSheetOpen} onOpenChange={setTargetSheetOpen}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-2xl">
          <form
            className="flex min-h-0 flex-1 flex-col"
            noValidate
            onSubmit={(event) => {
              event.preventDefault();
              void saveTarget();
            }}
          >
            <SheetHeader>
              <SheetTitle>Edit Target</SheetTitle>
            </SheetHeader>
            <div className="flex-1 overflow-y-auto px-4 pb-4">
              <MutationError title="Failed to Save Include" message={targetMutationError} />
              <MunkiSoftwareTargetFormFields
                form={targetForm}
                packages={packages}
                showErrors={showTargetErrors}
                errors={targetFormErrors}
                loadingPackages={false}
                className="max-w-none"
                onChange={setTargetForm}
              />
            </div>
            <SheetFooter className="border-t">
              <Button type="submit" size="sm">
                Save
              </Button>
              <Button type="button" variant="outline" size="sm" onClick={() => setTargetSheetOpen(false)}>
                Cancel
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>
    </PageShell>
  );
}

function excludeLabelIDsFromTargets(software: MunkiSoftwareDetail) {
  return software.targets.exclude.map((target) => target.label_id);
}

function targetRowsFromIncludes(includes: SoftwareInclude[]): MunkiSoftwareTargetRow[] {
  return includes.map((include, index) => ({ ...include, id: index + 1, priority: index + 1 }));
}

function newDraftTarget(priority: number, id: number): MunkiSoftwareTargetRow {
  return {
    id,
    priority,
    label_id: null,
    package: { strategy: "latest" },
    state: "managed_install",
    featured: false,
  };
}

function numberTargetRows(rows: MunkiSoftwareTargetRow[]) {
  return rows.map((row, index) => ({ ...row, priority: index + 1 }));
}

function munkiSoftwareTargetFormFromRow(row: MunkiSoftwareTargetRow): MunkiSoftwareTargetFormState {
  return {
    priority: row.priority,
    label_id: row.label_id,
    strategy: row.package.strategy,
    package_id: row.package.package_id ? String(row.package.package_id) : "",
    state: row.state,
    featured: row.featured,
  };
}

function targetIncludeMutations(rows: MunkiSoftwareTargetRow[]) {
  const includes: SoftwareInclude[] = [];
  const labelErrors: Partial<Record<number, string>> = {};
  let editRow: MunkiSoftwareTargetRow | null = null;
  for (const [index, row] of rows.entries()) {
    const form = { ...munkiSoftwareTargetFormFromRow(row), priority: index + 1 };
    const parsed = munkiSoftwareTargetFormSchema.safeParse(form);
    if (!parsed.success) {
      const rowErrors = fieldErrors(parsed);
      if (rowErrors.label_id) {
        labelErrors[row.id] = rowErrors.label_id;
      } else {
        editRow ??= row;
      }
      continue;
    }
    includes.push(munkiSoftwareInclude(parsed.data));
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

function updateRowLabel(row: MunkiSoftwareTargetRow, id: number, labelID: number | null): MunkiSoftwareTargetRow {
  if (row.id !== id) return row;
  return { ...row, label_id: labelID };
}

function sameTargetRows(a: MunkiSoftwareTargetRow[], b: SoftwareInclude[]) {
  if (a.length !== b.length) return false;
  return a.every((row, index) => sameTargetRow(row, b[index], index + 1));
}

function sameTargetRow(a: MunkiSoftwareTargetRow, b: SoftwareInclude | undefined, priority: number) {
  if (!b) return false;
  return (
    a.label_id === b.label_id &&
    a.priority === priority &&
    a.state === b.state &&
    a.featured === b.featured &&
    a.package.strategy === b.package.strategy &&
    (a.package.package_id ?? null) === (b.package.package_id ?? null)
  );
}

function targetPackageLabel(target: MunkiSoftwareTargetRow, packages: MunkiPackage[]) {
  if (target.package.strategy === "specific") {
    const pkg = packages.find((item) => item.id === target.package.package_id);
    return pkg ? `${pkg.software_name} ${pkg.version}` : munkiPackageStrategyLabel(target.package.strategy);
  }
  return munkiPackageStrategyLabel(target.package.strategy);
}

function targetAvailabilitySections(target: MunkiSoftwareTargetRow) {
  const sections: string[] = [];
  if (target.state === "optional_install") sections.push("Optional Installs");
  if (target.featured) sections.push("Featured Items");
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
