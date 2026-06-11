import { encodeSort } from "@/hooks/use-data-table-search";
import { useForm } from "@tanstack/react-form";
import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Info, Plus } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { EmptyPanel } from "@/components/empty-panel";
import { FormField } from "@/components/form-field";
import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { FreeTextCombobox } from "@/components/munki/free-text-combobox";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { QueryError } from "@/components/query-error";
import { SubmitButton } from "@/components/submit-button";
import { LabelAssignmentList } from "@/components/targeting/label-assignment-list";
import { Button } from "@/components/ui/button";
import { FieldError, FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
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
import { uniqueOptions } from "@/lib/form-validation";
import { formatRelative } from "@/lib/utils";

import { munkiSoftwareFormFromSoftware, munkiSoftwareInclude, munkiSoftwareSchema } from "./fields";
import { MunkiIncludeTargets, type MunkiSoftwareTargetRow } from "./include-targets";
import { munkiInstallerTypeLabel } from "./munki-software";

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
  const titles = useMunkiSoftware({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const updateSoftware = useUpdateMunkiSoftware();
  const iconUpload = useUploadMunkiArtifact("icon");
  const [targetRows, setTargetRows] = useState<MunkiSoftwareTargetRow[]>(() =>
    targetRowsFromIncludes(software.targets.include),
  );
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const [excludeForm, setExcludeForm] = useState<number[]>(() => excludeLabelIDsFromTargets(software));
  const packages = software.packages ?? [];
  const includes = useMemo(() => software.targets.include, [software.targets.include]);
  const excludeLabelIDs = useMemo(() => excludeLabelIDsFromTargets(software), [software]);
  const includeLabelIDs = useMemo(
    () => targetRows.flatMap((row) => (row.label_id === null ? [] : [row.label_id])),
    [targetRows],
  );
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

  function changeTargets(next: MunkiSoftwareTargetRow[]) {
    updateSoftware.reset();
    setTargetRows(numberTargetRows(next));
  }

  function changeExclude(next: number[]) {
    updateSoftware.reset();
    setExcludeForm(next);
  }

  async function savePage() {
    await softwareOptionsForm.handleSubmit();
    const softwareParsed = munkiSoftwareSchema.safeParse(softwareOptionsForm.state.values);
    if (!softwareParsed.success) return;

    const iconArtifact = iconFile ? await iconUpload.upload(iconFile) : null;
    const body: MunkiSoftwareMutation = {
      ...softwareParsed.data,
      icon_artifact_id: iconArtifact?.id ?? (iconCleared ? undefined : software.icon_artifact_id),
      targets: {
        include: targetRows.map(munkiSoftwareInclude),
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
    setExcludeForm(excludeLabelIDs);
  }

  const packageColumns: ColumnDef<MunkiPackage>[] = [
    {
      id: "version",
      accessorKey: "version",
      header: () => "Version",
      cell: ({ row }) => (
        <Link
          to="/munki/packages/$packageId/edit"
          params={{ packageId: String(row.original.id) }}
          className="flex min-w-0 items-center gap-3 hover:underline"
        >
          <PackageIconView pkg={row.original} />
          <span className="truncate font-medium">{row.original.version}</span>
        </Link>
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
                  <MunkiIncludeTargets
                    rows={targetRows}
                    excludeLabelIDs={excludeForm}
                    packages={packages}
                    onChange={changeTargets}
                  />
                  <Separator />
                  <LabelAssignmentList
                    title="Exclude"
                    addLabel="Add Exclude"
                    emptyText="No excludes yet"
                    rows={excludeForm.map((label_id) => ({ label_id }))}
                    crossListLabelIDs={includeLabelIDs}
                    includeBuiltins={false}
                    onChange={(rows) => changeExclude(rows.map((row) => row.label_id))}
                  />
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
                    <DataTableStatic columns={packageColumns} data={packages} />
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

function numberTargetRows(rows: MunkiSoftwareTargetRow[]) {
  return rows.map((row, index) => ({ ...row, priority: index + 1 }));
}

function PackageIconView({ pkg }: { pkg: MunkiPackage }) {
  return <MunkiIcon iconUrl={pkg.icon_url} size="md" />;
}
