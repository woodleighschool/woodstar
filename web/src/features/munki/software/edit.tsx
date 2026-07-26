import { useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Plus } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTableStatic } from "@components/data-table/data-table-static";
import { encodeSort } from "@components/data-table/use-data-table-search";
import { FormActions } from "@components/form-actions";
import {
  type FormTabDefinition,
  FormTabTrigger,
  revealFirstInvalidFormTab,
} from "@components/form-tabs";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@components/layout/scrollable-tabs";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryGate } from "@components/query-gate";
import { LabelAssignmentList } from "@components/targeting/label-assignment-list";
import { Button } from "@components/ui/button";
import { TabsContent, TabsTrigger } from "@components/ui/tabs";
import { ValidatedFormField } from "@components/validated-form-field";
import { SoftwareArtwork } from "@features/software/software-icon";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { MunkiPackage, MunkiSoftwareDetail } from "@lib/api";
import { uniqueOptions } from "@lib/form-validation";
import { MAX_PAGE_SIZE } from "@lib/pagination";
import { parseRouteID } from "@lib/route-params";
import { formatRelative } from "@lib/utils";

import {
  munkiSoftwareFormFromSoftware,
  munkiSoftwareInclude,
  MunkiSoftwareOptionsFields,
  munkiSoftwareSchema,
  useMunkiSoftwareForm,
} from "./fields";
import { MunkiIncludeTargets, type MunkiSoftwareTargetRow } from "./include-targets";
import { useMunkiSoftware, useMunkiSoftwareDetail, useUpdateMunkiSoftware } from "./queries";
import { useUploadMunkiIcon } from "./queries";
const softwareFormTabs = [
  {
    value: "options",
    fields: ["name", "display_name", "description", "category", "developer", "icon"],
  },
  { value: "targets", fields: ["targets"] },
] as const satisfies readonly FormTabDefinition[];

const packageColumns: ColumnDef<MunkiPackage>[] = [
  {
    id: "version",
    accessorKey: "version",
    header: () => "Version",
    cell: ({ row }) => (
      <div className="flex min-w-0 items-center gap-3">
        <SoftwareArtwork src={row.original.software.icon_url} size="md" />
        <Link
          to="/munki/packages/$id"
          params={{ id: String(row.original.id) }}
          className="min-w-0 truncate font-medium"
        >
          {row.original.version}
        </Link>
      </div>
    ),
  },
  {
    id: "installer_type",
    header: "Installer",
    enableSorting: false,
    cell: ({ row }) => row.original.installer_type,
  },
  {
    id: "updated_at",
    accessorKey: "updated_at",
    header: "Updated",
    enableSorting: false,
    cell: ({ row }) => formatRelative(row.original.updated_at),
  },
];

export function MunkiSoftwareEditPage() {
  const params = useParams({ strict: false });
  const softwareID = parseRouteID(params.id);
  const query = useMunkiSoftwareDetail(softwareID);
  if (softwareID === null) {
    return (
      <QueryGate
        title="Failed to load software"
        error={{ message: "Software route is invalid." }}
      />
    );
  }
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to load software"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  return (
    <MunkiSoftwareDetailForm
      key={`${query.data.id}:${query.data.updated_at}`}
      software={query.data}
    />
  );
}
function MunkiSoftwareDetailForm({ software }: { software: MunkiSoftwareDetail }) {
  const navigate = useNavigate();
  const titles = useMunkiSoftware({
    per_page: MAX_PAGE_SIZE,
    sort: encodeSort("name"),
  });
  const updateSoftware = useUpdateMunkiSoftware();
  const iconUpload = useUploadMunkiIcon();
  const [activeTab, setActiveTab] = useState("options");
  const packages = software.packages ?? [];
  const softwareOptionsForm = useMunkiSoftwareForm(
    munkiSoftwareFormFromSoftware(software),
    async (value) => {
      const data = munkiSoftwareSchema.parse(value);
      await updateSoftware.mutateAsync({
        id: software.id,
        body: {
          display_name: data.display_name,
          description: data.description,
          category: data.category,
          developer: data.developer,
          icon_object_id:
            value.icon.kind === "stored"
              ? value.icon.objectID
              : value.icon.kind === "upload"
                ? (software.icon_object_id ?? undefined)
                : undefined,
          targets: {
            include: value.targets.include.map(munkiSoftwareInclude),
            exclude: value.targets.exclude,
          },
        },
      });
      if (value.icon.kind === "upload") {
        await iconUpload.upload({
          softwareId: software.id,
          file: value.icon.file,
        });
      }
      return software.id;
    },
    (savedID) =>
      void navigate({
        to: "/munki/software/$id",
        params: { id: String(savedID) },
      }),
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
  function changeTargets(next: MunkiSoftwareTargetRow[]) {
    updateSoftware.reset();
    softwareOptionsForm.setFieldValue("targets.include", numberTargetRows(next));
  }
  function changeExclude(
    next: {
      label_id: number;
    }[],
  ) {
    updateSoftware.reset();
    softwareOptionsForm.setFieldValue("targets.exclude", next);
  }
  const exitGuard = usePageFormExitGuard({
    form: softwareOptionsForm,
    onDiscard: () =>
      void navigate({
        to: "/munki/software/$id",
        params: { id: String(software.id) },
      }),
  });
  const tabs = [
    {
      value: "options",
      label: "Options",
      content: (
        <MunkiSoftwareOptionsFields
          form={softwareOptionsForm}
          nameReadOnly
          categoryOptions={categoryOptions}
          developerOptions={developerOptions}
        />
      ),
    },
    {
      value: "targets",
      label: "Targets",
      content: (
        <softwareOptionsForm.Field name="targets">
          {(field) => {
            const includeLabelIDs = field.state.value.include.flatMap((row) =>
              row.label_id === null ? [] : [row.label_id],
            );
            const excludeLabelIDs = field.state.value.exclude.map((row) => row.label_id);
            return (
              <ValidatedFormField field={field}>
                {(control) => (
                  <div {...control} tabIndex={-1} className="flex flex-col gap-6">
                    <MunkiIncludeTargets
                      rows={field.state.value.include}
                      excludeLabelIDs={excludeLabelIDs}
                      packages={packages}
                      onChange={changeTargets}
                    />
                    <LabelAssignmentList
                      title="Exclude"
                      addLabel="Add Exclude"
                      emptyText="No excludes yet"
                      rows={field.state.value.exclude}
                      crossListLabelIDs={includeLabelIDs}
                      includeBuiltins={false}
                      onChange={changeExclude}
                    />
                  </div>
                )}
              </ValidatedFormField>
            );
          }}
        </softwareOptionsForm.Field>
      ),
    },
    {
      value: "packages",
      label: "Packages",
      content: (
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-base font-semibold">Packages</h2>
            <Button
              size="sm"
              variant="outline"
              render={<Link to="/munki/packages/new" search={{ software_id: software.id }} />}
              nativeButton={false}
            >
              <Plus data-icon="inline-start" />
              Add Package
            </Button>
          </div>
          {packages.length > 0 ? (
            <DataTableStatic columns={packageColumns} data={packages} />
          ) : (
            <PanelEmptyState>No packages yet</PanelEmptyState>
          )}
        </div>
      ),
    },
  ];
  return (
    <PageShell>
      <PageHeader title="Edit Software" />

      <form
        className="flex flex-col gap-5"
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void softwareOptionsForm.handleSubmit().then(() => {
            revealFirstInvalidFormTab(softwareOptionsForm, softwareFormTabs, setActiveTab);
            return undefined;
          });
        }}
      >
        <ScrollableTabs value={activeTab} onValueChange={setActiveTab}>
          <ScrollableTabsList>
            <FormTabTrigger form={softwareOptionsForm} tab={softwareFormTabs[0]}>
              Options
            </FormTabTrigger>
            <FormTabTrigger form={softwareOptionsForm} tab={softwareFormTabs[1]}>
              Targets
            </FormTabTrigger>
            <TabsTrigger value="packages">Packages</TabsTrigger>
          </ScrollableTabsList>
          {tabs.map((tab) => (
            <TabsContent
              key={tab.value}
              value={tab.value}
              keepMounted={tab.value === "packages" ? undefined : true}
              className="data-inactive:hidden"
            >
              {tab.content}
            </TabsContent>
          ))}
        </ScrollableTabs>

        <FormActions
          form={softwareOptionsForm}
          submitLabel="Save"
          onCancel={exitGuard.requestDiscard}
        />
        {exitGuard.dialog}
      </form>
    </PageShell>
  );
}
function numberTargetRows(rows: MunkiSoftwareTargetRow[]) {
  return rows.map((row, index) => ({ ...row, priority: index + 1 }));
}
