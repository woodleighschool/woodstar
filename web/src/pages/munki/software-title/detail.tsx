import { useForm } from "@tanstack/react-form";
import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, PackageCheck, Plus } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { DataTable, DataTableColumnHeader, DataTableEmptyState, DataTableRowDragHandle } from "@/components/data-table";
import type { LabelChip } from "@/components/labels/label-chip-utils";
import { LabelPicker } from "@/components/labels/label-picker";
import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { MutationError } from "@/components/mutation-error";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { FreeTextCombobox } from "@/components/ui/free-text-combobox";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useUploadMunkiArtifact } from "@/hooks/munki/artifacts";
import {
  useCreateMunkiAssignment,
  useReorderMunkiAssignments,
  useUpdateMunkiAssignment,
  useUpdateMunkiAssignmentExcludeLabels,
  type MunkiAssignment,
} from "@/hooks/munki/assignments";
import { type MunkiPackage } from "@/hooks/munki/packages";
import {
  useMunkiSoftwareTitle,
  useMunkiSoftwareTitles,
  useUpdateMunkiSoftwareTitle,
  type MunkiSoftwareTitleDetail,
  type MunkiSoftwareTitleMutation,
} from "@/hooks/munki/software-titles";
import { useLabels } from "@/hooks/use-labels";
import { fieldErrors } from "@/lib/form-validation";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative } from "@/lib/utils";

import { Sheet, SheetContent, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { MunkiAssignmentFormFields } from "./assignment-form";
import {
  emptyMunkiAssignmentForm,
  munkiAssignmentFormFromAssignment,
  munkiAssignmentFormSchema,
  munkiAssignmentMutation,
} from "./assignment-form-model";
import { softwareTitleFormFromTitle, softwareTitleSchema } from "./form-model";
import {
  munkiAssignmentActionLabel,
  munkiInstallerTypeLabel,
  munkiPackageSelectionLabel,
  munkiRestartActionLabel,
} from "./shared";
import { uniqueOptions } from "./utils";

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
  const labels = useLabels({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const updateSoftware = useUpdateMunkiSoftwareTitle();
  const iconUpload = useUploadMunkiArtifact("icon");
  const reorder = useReorderMunkiAssignments();
  const createAssignment = useCreateMunkiAssignment();
  const updateAssignment = useUpdateMunkiAssignment();
  const updateExcludes = useUpdateMunkiAssignmentExcludeLabels();
  const [reorderEnabled, setReorderEnabled] = useState(false);
  const [orderedIncludes, setOrderedIncludes] = useState<MunkiAssignment[]>([]);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const [assignmentSheetOpen, setAssignmentSheetOpen] = useState(false);
  const [editingAssignment, setEditingAssignment] = useState<MunkiAssignment | null>(null);
  const [assignmentForm, setAssignmentForm] = useState(() => emptyMunkiAssignmentForm());
  const [showAssignmentErrors, setShowAssignmentErrors] = useState(false);
  const [excludeForm, setExcludeForm] = useState<number[]>([]);
  const packages = software.packages ?? [];
  const includes = useMemo(() => software.includes ?? [], [software.includes]);
  const excludeLabelIDs = useMemo(() => software.exclude_label_ids ?? [], [software.exclude_label_ids]);
  const includeLabelIDs = useMemo(() => includes.map((assignment) => assignment.label_id), [includes]);
  const includeIDs = useMemo(() => includes.map((assignment) => assignment.id), [includes]);
  const orderedIncludeIDs = useMemo(() => orderedIncludes.map((assignment) => assignment.id), [orderedIncludes]);
  const softwareOptionsForm = useForm({
    defaultValues: softwareTitleFormFromTitle(software),
    validators: {
      onSubmit: softwareTitleSchema,
    },
    onSubmit: async ({ value }) => {
      const data = softwareTitleSchema.parse(value);
      const iconArtifact = iconFile ? await iconUpload.upload(iconFile) : null;
      const body: MunkiSoftwareTitleMutation = {
        ...data,
        icon_artifact_id: iconArtifact?.id ?? (iconCleared ? undefined : software.icon_artifact_id),
      };
      await updateSoftware.mutateAsync({ id: software.id, body });
      setIconFile(null);
      setIconCleared(false);
    },
  });
  const assignmentParsed = useMemo(() => munkiAssignmentFormSchema.safeParse(assignmentForm), [assignmentForm]);
  const assignmentErrors = useMemo(() => fieldErrors(assignmentParsed), [assignmentParsed]);
  const assignmentPending = createAssignment.isPending || updateAssignment.isPending;
  const assignmentMutationError = createAssignment.error?.message ?? updateAssignment.error?.message;
  const excludeDirty = !sameNumberSet(excludeForm, excludeLabelIDs);
  const orderDirty = !sameNumberSequence(orderedIncludeIDs, includeIDs);
  const pagePending =
    updateSoftware.isPending || iconUpload.isUploading || updateExcludes.isPending || reorder.isPending;
  const pageMutationError =
    updateSoftware.error?.message ??
    iconUpload.error?.message ??
    updateExcludes.error?.message ??
    reorder.error?.message;
  const assignmentUnavailableLabelIDs = useMemo(
    () => [...excludeForm, ...includeLabelIDs.filter((labelID) => labelID !== editingAssignment?.label_id)],
    [editingAssignment?.label_id, excludeForm, includeLabelIDs],
  );
  const labelsByID = useMemo<ReadonlyMap<number, LabelChip>>(
    () => new Map((labels.data?.items ?? []).map((label) => [label.id, label])),
    [labels.data?.items],
  );
  const categoryOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.category)),
    [titles.data?.items],
  );
  const developerOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.developer)),
    [titles.data?.items],
  );
  const title = software.display_name || software.name || "Software";

  useEffect(() => {
    setOrderedIncludes(includes);
  }, [includes]);

  useEffect(() => {
    setExcludeForm(excludeLabelIDs);
  }, [excludeLabelIDs]);

  function moveAssignments(next: MunkiAssignment[]) {
    reorder.reset();
    setOrderedIncludes(next.map((assignment, index) => ({ ...assignment, priority: index + 1 })));
  }

  function openNewAssignment() {
    createAssignment.reset();
    updateAssignment.reset();
    setEditingAssignment(null);
    setAssignmentForm(emptyMunkiAssignmentForm(includes.length + 1));
    setShowAssignmentErrors(false);
    setAssignmentSheetOpen(true);
  }

  function openEditAssignment(assignment: MunkiAssignment) {
    createAssignment.reset();
    updateAssignment.reset();
    setEditingAssignment(assignment);
    setAssignmentForm(munkiAssignmentFormFromAssignment(assignment));
    setShowAssignmentErrors(false);
    setAssignmentSheetOpen(true);
  }

  async function saveAssignment() {
    const next = munkiAssignmentFormSchema.safeParse(assignmentForm);
    if (!next.success) {
      setShowAssignmentErrors(true);
      return;
    }

    const body = munkiAssignmentMutation(software.id, next.data);
    if (editingAssignment) {
      await updateAssignment.mutateAsync({ id: editingAssignment.id, body });
    } else {
      await createAssignment.mutateAsync(body);
    }
    setAssignmentSheetOpen(false);
  }

  async function savePage() {
    const softwareDirty = iconFile !== null || iconCleared || softwareOptionsForm.state.isDirty;
    if (softwareDirty) {
      await softwareOptionsForm.handleSubmit();
      if (!softwareTitleSchema.safeParse(softwareOptionsForm.state.values).success) return;
    }
    if (orderDirty) {
      await reorder.mutateAsync({ softwareId: software.id, orderedIds: orderedIncludeIDs });
    }
    if (excludeDirty) {
      await updateExcludes.mutateAsync({ softwareId: software.id, excludeLabelIds: excludeForm });
    }
    await refetchSoftware();
    setReorderEnabled(false);
  }

  function resetAssignmentPage() {
    reorder.reset();
    updateExcludes.reset();
    updateSoftware.reset();
    iconUpload.reset();
    softwareOptionsForm.reset(softwareTitleFormFromTitle(software));
    setIconFile(null);
    setIconCleared(false);
    setOrderedIncludes(includes);
    setExcludeForm(excludeLabelIDs);
    setReorderEnabled(false);
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
            <div className="text-muted-foreground truncate text-xs">
              {row.original.display_name || row.original.name}
            </div>
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

  const includeColumns: ColumnDef<MunkiAssignment>[] = [
    ...(reorderEnabled
      ? ([
          {
            id: "drag",
            header: () => null,
            enableSorting: false,
            enableHiding: false,
            cell: () => <DataTableRowDragHandle />,
            meta: { headClassName: "w-10", cellClassName: "w-10" },
          },
        ] satisfies ColumnDef<MunkiAssignment>[])
      : []),
    {
      id: "priority",
      accessorKey: "priority",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Priority" />,
      cell: ({ row }) => row.original.priority,
      meta: { headClassName: "w-20", cellClassName: "w-20" },
    },
    {
      id: "label",
      accessorKey: "label_id",
      header: "Label",
      enableSorting: false,
      cell: ({ row }) => labelsByID.get(row.original.label_id)?.name ?? `Label ${row.original.label_id}`,
    },
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
                    name="display_name"
                    children={(field) => {
                      const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                      return (
                        <Field data-invalid={invalid}>
                          <FieldLabel htmlFor="munki-software-display-name">Display Name</FieldLabel>
                          <Input
                            id="munki-software-display-name"
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
                  <softwareOptionsForm.Field
                    name="description"
                    children={(field) => {
                      const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                      return (
                        <Field data-invalid={invalid}>
                          <FieldLabel htmlFor="munki-software-description">Description</FieldLabel>
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
                        <Button type="button" size="sm" onClick={openNewAssignment}>
                          <Plus data-icon="inline-start" />
                          Add Target
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          disabled={reorderEnabled || orderedIncludes.length < 2}
                          onClick={() => setReorderEnabled(true)}
                        >
                          Order
                        </Button>
                        {reorderEnabled ? (
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            disabled={pagePending}
                            onClick={resetAssignmentPage}
                          >
                            Cancel Order
                          </Button>
                        ) : null}
                      </div>
                    </div>
                    <DataTable
                      columns={includeColumns}
                      data={reorderEnabled ? orderedIncludes : includes}
                      totalCount={includes.length}
                      pagination={{ pageIndex: 0, pageSize: Math.max(includes.length, 1) }}
                      sorting={[]}
                      onPaginationChange={() => undefined}
                      onSortingChange={() => undefined}
                      isLoading={false}
                      clientSort
                      onRowClick={
                        reorderEnabled
                          ? undefined
                          : (row) => {
                              openEditAssignment(row);
                            }
                      }
                      rowReorderDisabled={!reorderEnabled || reorder.isPending || orderedIncludes.length <= 1}
                      onRowReorder={moveAssignments}
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
                        updateExcludes.reset();
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
            const pageDirty = softwareDirty || excludeDirty || orderDirty;

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
              <SheetTitle>{editingAssignment ? "Edit Include" : "New Include"}</SheetTitle>
            </SheetHeader>
            <div className="flex-1 overflow-y-auto px-4 pb-4">
              <MutationError title="Failed to Save Include" message={assignmentMutationError} />
              <MunkiAssignmentFormFields
                form={assignmentForm}
                packages={packages}
                showErrors={showAssignmentErrors}
                errors={assignmentErrors}
                loadingPackages={false}
                unavailableLabelIDs={assignmentUnavailableLabelIDs}
                className="max-w-none"
                onChange={setAssignmentForm}
              />
            </div>
            <SheetFooter className="border-t">
              <Button type="submit" size="sm" disabled={assignmentPending}>
                {assignmentPending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
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

function assignmentPackageLabel(assignment: MunkiAssignment) {
  if (assignment.package_selection === "specific_package") {
    return assignment.pinned_package_version
      ? `${assignment.pinned_package_name ?? "Pinned"} ${assignment.pinned_package_version}`
      : munkiPackageSelectionLabel(assignment.package_selection);
  }
  return munkiPackageSelectionLabel(assignment.package_selection);
}

function assignmentActionLabel(assignment: MunkiAssignment) {
  return munkiAssignmentActionLabel(assignment.action);
}

function assignmentMSCSections(assignment: MunkiAssignment) {
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

function sameNumberSequence(a: readonly number[], b: readonly number[]) {
  return a.length === b.length && a.every((value, index) => value === b[index]);
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
    pkg.uninstallable ? "Uninstallable" : "",
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
