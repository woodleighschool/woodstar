import { useForm } from "@tanstack/react-form";
import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Info, Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTable, DataTableColumnHeader } from "@/components/data-table";
import { EmptyPanel } from "@/components/empty-panel";
import { FormField } from "@/components/form-field";
import { LabelPicker } from "@/components/labels/label-picker";
import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { FreeTextCombobox } from "@/components/munki/free-text-combobox";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { QueryError } from "@/components/query-error";
import { SubmitButton } from "@/components/submit-button";
import { LabelTargetRowsTable } from "@/components/targeting/label-target-rows-table";
import { TargetSection } from "@/components/targeting/target-section";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
} from "@/components/ui/combobox";
import { Field, FieldError, FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useUploadMunkiArtifact } from "@/hooks/use-munki-artifacts";
import { type MunkiPackage } from "@/hooks/use-munki-packages";
import {
  useMunkiSoftware,
  useMunkiSoftwareDetail,
  useUpdateMunkiSoftware,
  type MunkiSoftwareDetail,
  type MunkiSoftwareMutation,
} from "@/hooks/use-munki-software";
import type { SoftwareInclude } from "@/lib/api";
import { fieldErrors, uniqueOptions } from "@/lib/form-validation";
import { selectedLabelTargetIDs } from "@/lib/label-target-rows";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative } from "@/lib/utils";
import {
  LATEST_PACKAGE_VALUE,
  munkiSoftwareFormFromSoftware,
  munkiSoftwareInclude,
  munkiSoftwareSchema,
  munkiSoftwareTargetSchema,
  targetPackageFromValue,
  targetPackageValue,
} from "./fields";

import { MUNKI_SOFTWARE_ACTION_OPTIONS, munkiInstallerTypeLabel } from "./munki-software";

interface MunkiSoftwareTargetRow {
  id: number;
  label_id: number | null;
  priority: number;
  package: SoftwareInclude["package"];
  actions: SoftwareInclude["actions"];
}

interface MunkiSoftwareTargetRowErrors {
  label_id?: string;
  package?: string;
  actions?: string;
}

export function MunkiSoftwareEditPage() {
  const params = useParams({ strict: false });
  const softwareId = Number(params.softwareId);
  const softwareID = Number.isFinite(softwareId) && softwareId > 0 ? softwareId : null;
  const query = useMunkiSoftwareDetail(softwareID);

  if (softwareID === null) {
    return (
      <PageShell>
        <QueryError title="Failed to load software" error={{ message: "Software route is invalid." }} />
      </PageShell>
    );
  }

  if (query.error) {
    return (
      <PageShell>
        <QueryError title="Failed to load software" error={query.error} onRetry={() => void query.refetch()} />
      </PageShell>
    );
  }

  if (!query.data) return null;

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
  const titles = useMunkiSoftware({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const updateSoftware = useUpdateMunkiSoftware();
  const iconUpload = useUploadMunkiArtifact("icon");
  const [targetRows, setTargetRows] = useState<MunkiSoftwareTargetRow[]>(() =>
    targetRowsFromIncludes(software.targets.include),
  );
  const [nextDraftID, setNextDraftID] = useState(-1);
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const [targetErrors, setTargetErrors] = useState<Partial<Record<number, MunkiSoftwareTargetRowErrors>>>({});
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
  const pagePending = updateSoftware.isPending || iconUpload.isUploading;
  const pageError = updateSoftware.error?.message ?? iconUpload.error?.message;

  // Category/developer suggestions capped at MAX_PAGE_SIZE for now.
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

  function updateTargetRow(id: number, update: (target: MunkiSoftwareTargetRow) => MunkiSoftwareTargetRow) {
    updateSoftware.reset();
    setTargetErrors((current) => omitTargetError(current, id));
    setTargetRows((current) => current.map((row) => (row.id === id ? update(row) : row)));
  }

  function updateTargetLabel(id: number, labelID: number | null) {
    updateTargetRow(id, (row) => updateRowLabel(row, id, labelID));
  }

  function deleteTargetRow(target: MunkiSoftwareTargetRow) {
    updateSoftware.reset();
    setTargetRows((current) => numberTargetRows(current.filter((row) => row.id !== target.id)));
    setTargetErrors((current) => omitTargetError(current, target.id));
  }

  async function savePage() {
    await softwareOptionsForm.handleSubmit();
    const softwareParsed = munkiSoftwareSchema.safeParse(softwareOptionsForm.state.values);
    if (!softwareParsed.success) return;

    const nextTargets = targetIncludeMutations(targetRows);
    if (!nextTargets.success) {
      setTargetErrors(nextTargets.errors);
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
      cell: ({ row }) => (
        <TargetPackageSelect
          target={row.original}
          packages={packages}
          error={targetErrors[row.original.id]?.package}
          onChange={(selector) => updateTargetRow(row.original.id, (target) => ({ ...target, package: selector }))}
        />
      ),
      meta: { cellClassName: "min-w-44 align-top" },
    },
    {
      id: "actions",
      accessorKey: "actions",
      header: "Actions",
      enableSorting: false,
      cell: ({ row }) => (
        <TargetActionsPicker
          target={row.original}
          error={targetErrors[row.original.id]?.actions}
          onChange={(actions) => updateTargetRow(row.original.id, (target) => ({ ...target, actions }))}
        />
      ),
      meta: { cellClassName: "min-w-52 align-top" },
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
                      <softwareOptionsForm.Field name="name">
                        {(field) => (
                          <FormField field={field} label="Name" htmlFor="munki-software-name" required>
                            {(control) => (
                              <Input
                                {...control}
                                name={field.name}
                                value={field.state.value}
                                onBlur={field.handleBlur}
                                onChange={(event) => field.handleChange(event.target.value)}
                              />
                            )}
                          </FormField>
                        )}
                      </softwareOptionsForm.Field>
                    </div>
                  </div>
                  <softwareOptionsForm.Field name="description">
                    {(field) => (
                      <FormField
                        field={field}
                        htmlFor="munki-software-description"
                        label={
                          <>
                            Description
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button variant="ghost" size="icon-xs" type="button">
                                  <Info className="size-3.5" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Description is shown in Managed Software Center.</TooltipContent>
                            </Tooltip>
                          </>
                        }
                      >
                        {(control) => (
                          <Textarea
                            {...control}
                            name={field.name}
                            value={field.state.value}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                        )}
                      </FormField>
                    )}
                  </softwareOptionsForm.Field>
                  <div className="grid gap-4 md:grid-cols-2">
                    <softwareOptionsForm.Field name="category">
                      {(field) => (
                        <FormField field={field} label="Category" htmlFor="munki-software-category">
                          {(control) => (
                            <FreeTextCombobox
                              id={control.id}
                              name={field.name}
                              value={field.state.value}
                              options={categoryOptions}
                              invalid={control["aria-invalid"]}
                              onBlur={field.handleBlur}
                              onChange={field.handleChange}
                            />
                          )}
                        </FormField>
                      )}
                    </softwareOptionsForm.Field>
                    <softwareOptionsForm.Field name="developer">
                      {(field) => (
                        <FormField field={field} label="Developer" htmlFor="munki-software-developer">
                          {(control) => (
                            <FreeTextCombobox
                              id={control.id}
                              name={field.name}
                              value={field.state.value}
                              options={developerOptions}
                              invalid={control["aria-invalid"]}
                              onBlur={field.handleBlur}
                              onChange={field.handleChange}
                            />
                          )}
                        </FormField>
                      )}
                    </softwareOptionsForm.Field>
                  </div>
                </FieldGroup>
              ),
            },
            {
              value: "targets",
              label: "Targets",
              content: (
                <div className="flex flex-col gap-6">
                  <TargetSection
                    title="Include"
                    action={
                      <Button type="button" variant="outline" size="sm" disabled={pagePending} onClick={openNewTarget}>
                        <Plus data-icon="inline-start" />
                        Add Include
                      </Button>
                    }
                  >
                    <LabelTargetRowsTable
                      rows={targetRows}
                      excludeLabelIDs={excludeForm}
                      labelErrors={targetLabelErrors(targetErrors)}
                      columnsAfterLabel={targetDetailColumns}
                      onChange={moveTargets}
                      onLabelChange={updateTargetLabel}
                      renderActions={(row) => (
                        <Button type="button" variant="ghost" size="icon-sm" onClick={() => deleteTargetRow(row)}>
                          <Trash2 />
                        </Button>
                      )}
                      empty={<EmptyPanel>No targets yet</EmptyPanel>}
                    />
                  </TargetSection>
                  <TargetSection title="Exclude" contentClassName="max-w-3xl">
                    <LabelPicker
                      value={excludeForm}
                      includeBuiltins={false}
                      unavailableLabelIDs={includeLabelIDs}
                      placeholder="Add Exclude"
                      emptyMessage="No Labels Found."
                      onChange={(next) => {
                        updateSoftware.reset();
                        setExcludeForm(next);
                      }}
                    />
                  </TargetSection>
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
                      <Link to="/munki/packages/new" search={{ software_id: software.id }}>
                        <Plus data-icon="inline-start" />
                        Add Package
                      </Link>
                    </Button>
                  </div>
                  {packages.length > 0 ? (
                    <DataTable
                      columns={packageColumns}
                      data={packages}
                      totalCount={packages.length}
                      pagination={{ pageIndex: 0, pageSize: packages.length }}
                      sorting={[]}
                      onPaginationChange={() => undefined}
                      onSortingChange={() => undefined}
                      isLoading={false}
                      clientSort
                      rowHref={(row) => ({
                        to: "/munki/packages/$packageId/edit",
                        params: { packageId: String(row.id) },
                      })}
                      empty={null}
                    />
                  ) : (
                    <EmptyPanel>No packages yet</EmptyPanel>
                  )}
                </div>
              ),
            },
          ]}
        />

        <div className="flex flex-col gap-2 border-t pt-4">
          {pageError ? <FieldError>{pageError}</FieldError> : null}
          <div className="flex items-center gap-2">
            <SubmitButton pending={pagePending} size="sm">
              Save
            </SubmitButton>
            <Button type="button" variant="outline" size="sm" onClick={resetTargetPage}>
              Cancel
            </Button>
          </div>
        </div>
      </form>
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
    actions: ["managed_installs"],
  };
}

function numberTargetRows(rows: MunkiSoftwareTargetRow[]) {
  return rows.map((row, index) => ({ ...row, priority: index + 1 }));
}

function targetIncludeMutations(rows: MunkiSoftwareTargetRow[]) {
  const includes: SoftwareInclude[] = [];
  const errors: Partial<Record<number, MunkiSoftwareTargetRowErrors>> = {};
  for (const [index, row] of rows.entries()) {
    const parsed = munkiSoftwareTargetSchema.safeParse({ ...row, priority: index + 1 });
    if (!parsed.success) {
      errors[row.id] = fieldErrors(parsed);
      continue;
    }
    includes.push(munkiSoftwareInclude(parsed.data));
  }
  if (Object.keys(errors).length > 0) {
    return { success: false as const, errors };
  }
  return { success: true as const, includes };
}

function omitTargetError<T>(errors: Partial<Record<number, T>>, id: number) {
  const next = { ...errors };
  delete next[id];
  return next;
}

function updateRowLabel(row: MunkiSoftwareTargetRow, id: number, labelID: number | null): MunkiSoftwareTargetRow {
  if (row.id !== id) return row;
  return { ...row, label_id: labelID };
}

function targetLabelErrors(errors: Partial<Record<number, MunkiSoftwareTargetRowErrors>>) {
  const out: Partial<Record<number, string>> = {};
  for (const [id, error] of Object.entries(errors)) {
    if (error?.label_id) out[Number(id)] = error.label_id;
  }
  return out;
}

function TargetPackageSelect({
  target,
  packages,
  error,
  onChange,
}: {
  target: MunkiSoftwareTargetRow;
  packages: MunkiPackage[];
  error?: string;
  onChange: (selector: SoftwareInclude["package"]) => void;
}) {
  return (
    <Field data-invalid={error ? true : undefined} className="gap-1">
      <Select
        value={targetPackageValue(target.package)}
        onValueChange={(value) => onChange(targetPackageFromValue(value))}
      >
        <SelectTrigger className="w-full">
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
      {error ? <FieldError>{error}</FieldError> : null}
    </Field>
  );
}

function TargetActionsPicker({
  target,
  error,
  onChange,
}: {
  target: MunkiSoftwareTargetRow;
  error?: string;
  onChange: (actions: SoftwareInclude["actions"]) => void;
}) {
  const warning = targetActionWarning(target.actions);
  const selected = MUNKI_SOFTWARE_ACTION_OPTIONS.filter((option) => target.actions.includes(option.value));
  const visibleSelected = selected.slice(0, 2);
  const hiddenCount = selected.length - visibleSelected.length;

  return (
    <Field data-invalid={error ? true : undefined} className="gap-1">
      <Combobox
        items={MUNKI_SOFTWARE_ACTION_OPTIONS}
        multiple
        value={selected}
        itemToStringLabel={(option) => option.label}
        itemToStringValue={(option) => option.value}
        onValueChange={(next) => onChange(next.map((option) => option.value))}
      >
        <ComboboxChips className="h-9 min-h-9 flex-nowrap overflow-hidden">
          <ComboboxValue>
            {visibleSelected.map((option) => (
              <ComboboxChip key={option.value} className="min-w-0 max-w-[8.5rem]">
                <span className="min-w-0 truncate">{option.label}</span>
              </ComboboxChip>
            ))}
            {hiddenCount > 0 ? (
              <span className="flex h-[calc(--spacing(5.5))] shrink-0 items-center rounded-sm bg-muted px-1.5 text-xs font-medium text-foreground">
                +{hiddenCount}
              </span>
            ) : null}
          </ComboboxValue>
          <ComboboxChipsInput
            placeholder={selected.length === 0 ? "Pick actions" : ""}
            required={selected.length === 0}
            aria-invalid={error ? true : undefined}
            className="min-w-0"
          />
        </ComboboxChips>
        <ComboboxContent>
          <ComboboxEmpty>No Actions Found.</ComboboxEmpty>
          <ComboboxList>
            {(option: (typeof MUNKI_SOFTWARE_ACTION_OPTIONS)[number]) => (
              <ComboboxItem key={option.value} value={option}>
                <span className="min-w-0 flex-1 truncate">{option.label}</span>
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
      {error ? <FieldError>{error}</FieldError> : null}
      {warning ? <p className="text-warning text-xs">{warning}</p> : null}
    </Field>
  );
}

function targetActionWarning(actions: SoftwareInclude["actions"]) {
  if (actions.includes("featured_items") && !actions.includes("optional_installs")) {
    return "Munki ignores featured items unless the item is also optional.";
  }
  return "";
}

function PackageIconView({ pkg }: { pkg: MunkiPackage }) {
  return <MunkiIcon iconUrl={pkg.icon_url} size="md" />;
}
