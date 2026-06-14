import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Plus } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { EmptyPanel } from "@/components/empty-panel";
import { FormActions } from "@/components/form-actions";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { MunkiIcon } from "@/components/munki/munki-icon";
import { QueryError } from "@/components/query-error";
import { LabelAssignmentList } from "@/components/targeting/label-assignment-list";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { TabsContent, TabsTrigger } from "@/components/ui/tabs";
import { encodeSort, MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import { useUploadMunkiIcon } from "@/hooks/use-munki-uploads";
import { type MunkiPackage } from "@/hooks/use-munki-packages";
import {
  type MunkiSoftwareDetail,
  useMunkiSoftware,
  useMunkiSoftwareDetail,
  useUpdateMunkiSoftware,
} from "@/hooks/use-munki-software";
import type { SoftwareInclude } from "@/lib/api";
import { uniqueOptions } from "@/lib/form-validation";
import { formatRelative } from "@/lib/utils";

import {
  munkiSoftwareFormFromSoftware,
  munkiSoftwareInclude,
  MunkiSoftwareOptionsFields,
  munkiSoftwareSchema,
  useMunkiSoftwareForm,
} from "./fields";
import { MunkiIncludeTargets, type MunkiSoftwareTargetRow } from "./include-targets";

export function MunkiSoftwareEditPage() {
  const params = useParams({ strict: false });
  const softwareId = Number(params.softwareId);
  const softwareID = Number.isFinite(softwareId) && softwareId > 0 ? softwareId : null;
  const query = useMunkiSoftwareDetail(softwareID);

  if (softwareID === null) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load software"
          error={{ message: "Software route is invalid." }}
        />
      </PageShell>
    );
  }

  if (query.error) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load software"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
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
  const iconUpload = useUploadMunkiIcon();
  const [targetRows, setTargetRows] = useState<MunkiSoftwareTargetRow[]>(() =>
    targetRowsFromIncludes(software.targets.include),
  );
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const [excludeForm, setExcludeForm] = useState<number[]>(() =>
    excludeLabelIDsFromTargets(software),
  );
  const packages = software.packages ?? [];
  const includes = useMemo(() => software.targets.include, [software.targets.include]);
  const excludeLabelIDs = useMemo(() => excludeLabelIDsFromTargets(software), [software]);
  const includeLabelIDs = useMemo(
    () => targetRows.flatMap((row) => (row.label_id === null ? [] : [row.label_id])),
    [targetRows],
  );
  const softwareOptionsForm = useMunkiSoftwareForm(
    munkiSoftwareFormFromSoftware(software),
    async (value) => {
      const data = munkiSoftwareSchema.parse(value);
      await updateSoftware.mutateAsync({
        id: software.id,
        body: {
          ...data,
          icon_object_id: iconCleared ? undefined : (software.icon_object_id ?? undefined),
          targets: {
            include: targetRows.map(munkiSoftwareInclude),
            exclude: excludeForm.map((label_id) => ({ label_id })),
          },
        },
      });
      if (iconFile) {
        await iconUpload.upload({ softwareId: software.id, file: iconFile });
      }
      setIconFile(null);
      setIconCleared(false);
      await refetchSoftware();
    },
  );
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
      cell: ({ row }) => row.original.installer_type,
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
  const tabs = [
    {
      value: "options",
      label: "Options",
      content: (
        <MunkiSoftwareOptionsFields
          form={softwareOptionsForm}
          categoryOptions={categoryOptions}
          developerOptions={developerOptions}
          icon={{
            iconUrl: iconCleared ? undefined : software.icon_url,
            file: iconFile,
            clearable: !!iconFile || (!iconCleared && !!software.icon_object_id),
            onFileChange: (file) => {
              setIconFile(file);
              setIconCleared(false);
            },
            onClear: () => {
              setIconFile(null);
              setIconCleared(!!software.icon_object_id);
            },
          }}
        />
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
  ];

  return (
    <PageShell>
      <PageHeader title={title} />

      <form
        className="flex flex-col gap-5"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void softwareOptionsForm.handleSubmit();
        }}
      >
        <ScrollableTabs defaultValue="options">
          <ScrollableTabsList>
            {tabs.map((tab) => (
              <TabsTrigger key={tab.value} value={tab.value}>
                {tab.label}
              </TabsTrigger>
            ))}
          </ScrollableTabsList>
          {tabs.map((tab) => (
            <TabsContent key={tab.value} value={tab.value} className="min-w-0">
              {tab.content}
            </TabsContent>
          ))}
        </ScrollableTabs>

        <FormActions form={softwareOptionsForm} requireDirty={false} onCancel={resetTargetPage} />
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
