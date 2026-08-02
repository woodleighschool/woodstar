import { getRouteApi, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { AppWindow, Pencil, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTable } from "@components/data-table/data-table";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { DataTableStatic } from "@components/data-table/data-table-static";
import { TableSurface } from "@components/data-table/table-surface";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@components/layout/scrollable-tabs";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { QueryGate } from "@components/query-gate";
import { TargetDetails } from "@components/targeting/target-details";
import { Badge } from "@components/ui/badge";
import { Button } from "@components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { TabsContent, TabsTrigger } from "@components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@components/ui/tooltip";
import { useAuth } from "@features/auth/queries";
import { useLabelNameMap } from "@features/labels/components/label-ref-list";
import { SoftwareArtwork } from "@features/software/software-icon";
import type {
  MunkiInclude,
  MunkiDeploymentHost,
  MunkiPackage,
  MunkiPackageDeployment,
  MunkiSoftwareDetail,
} from "@lib/api";
import { parseRouteID } from "@lib/route-params";
import { formatDateTime, formatRelative } from "@lib/utils";

import { MUNKI_ASSIGNMENT_ACTION_VALUES, MUNKI_SOFTWARE_ACTIONS } from "./actions";
import { MunkiSoftwareDeleteDialog } from "./delete-dialog";
import {
  deploymentStatusName,
  MUNKI_DEPLOYMENT_STATUS_OPTIONS,
  MUNKI_REPORT_STATES,
} from "./deployment";
import { useMunkiSoftwareDetail, useMunkiSoftwareHosts } from "./queries";

const routeApi = getRouteApi("/_authenticated/munki/software/$id/");
const HOST_FILTER_KEYS = [{ id: "status" }, { id: "action" }] as const;
const ASSIGNMENT_OPTIONS = MUNKI_ASSIGNMENT_ACTION_VALUES.map((value) => ({
  value,
  label: MUNKI_SOFTWARE_ACTIONS[value].name,
}));

function packageColumns(
  deployments: ReadonlyMap<string, MunkiPackageDeployment>,
): ColumnDef<MunkiPackage>[] {
  return [
    {
      id: "version",
      accessorKey: "version",
      header: () => "Version",
      cell: ({ row }) => (
        <div className="flex min-w-0 items-center gap-2">
          <SoftwareArtwork src={row.original.software.icon_url} />
          <Link
            to="/munki/packages/$id"
            params={{ id: String(row.original.id) }}
            className="min-w-0 truncate font-medium"
            title={row.original.version}
          >
            {row.original.version}
          </Link>
        </div>
      ),
    },
    {
      id: "installer_type",
      accessorKey: "installer_type",
      header: () => "Installer",
    },
    {
      id: "installed",
      header: () => "Installed",
      cell: ({ row }) => (
        <PackageInstallationCount deployment={deployments.get(row.original.version)} />
      ),
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: () => "Updated",
      cell: ({ row }) => formatRelative(row.original.updated_at),
    },
  ];
}

const hostColumns: ColumnDef<MunkiDeploymentHost>[] = [
  {
    id: "display_name",
    accessorKey: "display_name",
    header: () => "Host",
    cell: ({ row }) => <AssignedHostName host={row.original} />,
  },
  {
    id: "action",
    header: () => "Assignment",
    enableSorting: false,
    enableColumnFilter: true,
    cell: ({ row }) => <HostAssignment host={row.original} />,
  },
  {
    id: "status",
    accessorKey: "status",
    header: () => "Status",
    enableColumnFilter: true,
    cell: ({ row }) => hostDeploymentStatusLabel(row.original),
  },
  {
    id: "installed_version",
    accessorKey: "installed_version",
    header: () => "Installed",
    cell: ({ row }) => installedVersionLabel(row.original),
  },
  {
    id: "target_version",
    accessorKey: "target_version",
    header: () => "Target",
    cell: ({ row }) => targetVersionLabel(row.original),
  },
  {
    id: "last_successful_at",
    accessorKey: "last_successful_at",
    header: () => "Last Collected",
    cell: ({ row }) => <AssignedHostLastCollected host={row.original} />,
  },
];

export function MunkiSoftwareDetailPage() {
  const params = useParams({ strict: false });
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const { user } = useAuth();
  const softwareID = parseRouteID(params.id);
  const query = useMunkiSoftwareDetail(softwareID);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const activeTab = search.tab ?? "overview";

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

  const software = query.data;
  const isAdmin = user?.role === "admin";
  const deployments = new Map(
    software.deployment.packages.map((deployment) => [deployment.version, deployment]),
  );
  return (
    <PageShell className="gap-6">
      <PageHeader
        title="Software Details"
        icon={<SoftwareArtwork src={software.icon_url} fallbackIcon={AppWindow} loading="eager" />}
        meta={`Edited ${formatRelative(software.updated_at)}`}
        actions={
          isAdmin ? (
            <>
              <Button
                size="sm"
                render={<Link to="/munki/software/$id/edit" params={{ id: String(software.id) }} />}
                nativeButton={false}
              >
                <Pencil data-icon="inline-start" />
                Edit
              </Button>
              <Button
                type="button"
                variant="destructive"
                size="sm"
                onClick={() => setDeleteOpen(true)}
              >
                <Trash2 data-icon="inline-start" />
                Delete
              </Button>
            </>
          ) : null
        }
      />

      <ScrollableTabs value={activeTab}>
        <ScrollableTabsList>
          <TabsTrigger
            value="overview"
            render={
              <Link
                to="/munki/software/$id"
                params={{ id: String(software.id) }}
                search={{ ...search, tab: undefined }}
              />
            }
            nativeButton={false}
          >
            Overview
          </TabsTrigger>
          <TabsTrigger
            value="targets"
            render={
              <Link
                to="/munki/software/$id"
                params={{ id: String(software.id) }}
                search={{ ...search, tab: "targets" }}
              />
            }
            nativeButton={false}
          >
            Targets
          </TabsTrigger>
          <TabsTrigger
            value="packages"
            render={
              <Link
                to="/munki/software/$id"
                params={{ id: String(software.id) }}
                search={{ ...search, tab: "packages" }}
              />
            }
            nativeButton={false}
          >
            Packages
          </TabsTrigger>
        </ScrollableTabsList>

        <TabsContent value="overview">
          <KeyValueSection title="Overview">
            <KeyValueRow label="Name" value={software.name} />
            <KeyValueRow label="Display Name" value={software.display_name || "-"} />
            <KeyValueRow label="Description" value={software.description} />
            <KeyValueRow label="Category" value={software.category} />
            <KeyValueRow label="Developer" value={software.developer} />
            <KeyValueRow label="Packages" value={software.packages.length} />
          </KeyValueSection>
        </TabsContent>

        <TabsContent value="targets">
          <div className="flex flex-col gap-6">
            <MunkiSoftwareTargets software={software} />
            <MunkiHostTable softwareID={software.id} enabled={activeTab === "targets"} />
          </div>
        </TabsContent>

        <TabsContent value="packages">
          <DataTableStatic
            columns={packageColumns(deployments)}
            data={software.packages}
            empty={<PanelEmptyState>No packages yet</PanelEmptyState>}
          />
        </TabsContent>
      </ScrollableTabs>

      <MunkiSoftwareDeleteDialog
        software={software}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() => void navigate({ to: "/munki/software" })}
      />
    </PageShell>
  );
}

function MunkiHostTable({ softwareID, enabled }: { softwareID: number; enabled: boolean }) {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search: {
      q: search.host_q,
      page: search.host_page,
      per_page: search.host_per_page,
      sort: search.host_sort,
      status: search.status,
      action: search.action,
    },
    onSearchChange: (updater) =>
      void navigate({
        search: (previous) => {
          const next = updater({
            q: previous.host_q,
            page: previous.host_page,
            per_page: previous.host_per_page,
            sort: previous.host_sort,
            status: previous.status,
            action: previous.action,
          });
          return {
            ...previous,
            host_q: next.q,
            host_page: next.page,
            host_per_page: next.per_page,
            host_sort: next.sort,
            status: next.status,
            action: next.action,
          };
        },
        replace: true,
      }),
    filterKeys: HOST_FILTER_KEYS,
  });
  const query = useMunkiSoftwareHosts(enabled ? softwareID : null, {
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    status: search.status,
    action: search.action,
  });
  const rows = query.data?.items ?? [];
  const rowCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(rowCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: rows,
    columns: hostColumns,
    pageCount,
    rowCount,
    getRowId: (row) => String(row.host_id),
  });

  if (query.error) {
    return (
      <section className="flex flex-col gap-3">
        <h2 className="text-base/snug font-medium text-foreground">Hosts</h2>
        <QueryError
          title="Failed to load hosts"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      </section>
    );
  }
  if (query.isLoading) {
    return <DataTableSkeleton columnCount={6} filterCount={2} withViewOptions={false} />;
  }
  return (
    <DataTable
      heading="Hosts"
      table={table}
      empty={
        <PanelEmptyState>
          {tableSearch.isFiltered ? "No matching hosts" : "No hosts"}
        </PanelEmptyState>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        <DataTableSearchInput
          value={tableSearch.q ?? ""}
          onValueChange={tableSearch.onQueryChange}
          placeholder="Search hosts"
          className="h-8 w-full sm:w-64"
        />
        <DataTableFacetedFilter
          column={table.getColumn("status")}
          title="Status"
          options={MUNKI_DEPLOYMENT_STATUS_OPTIONS}
        />
        <DataTableFacetedFilter
          column={table.getColumn("action")}
          title="Assignment"
          options={ASSIGNMENT_OPTIONS}
        />
        {tableSearch.isFiltered ? (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => tableSearch.clearSearchKeys(["q", "status", "action"])}
          >
            Clear filters
          </Button>
        ) : null}
      </div>
    </DataTable>
  );
}

function AssignedHostName({ host }: { host: MunkiDeploymentHost }) {
  const link = (
    <Link to="/hosts/$id" params={{ id: String(host.host_id) }} className="font-medium">
      {host.display_name}
    </Link>
  );
  if (!host.hardware_serial) return link;
  return (
    <Tooltip>
      <TooltipTrigger render={link} />
      <TooltipContent>{host.hardware_serial}</TooltipContent>
    </Tooltip>
  );
}

function HostAssignment({ host }: { host: MunkiDeploymentHost }) {
  const actions = MUNKI_ASSIGNMENT_ACTION_VALUES.filter((action) => host.actions.includes(action));
  return (
    <div>
      <div>{actions.map((action) => MUNKI_SOFTWARE_ACTIONS[action].name).join(", ")}</div>
      <div className="text-xs text-muted-foreground">{assignedPackageLabel(host)}</div>
    </div>
  );
}

function assignedPackageLabel(host: MunkiDeploymentHost) {
  return host.package.strategy === "specific" ? host.package.version : "Latest";
}

function installedVersionLabel(host: MunkiDeploymentHost) {
  if (host.report_state !== "current") return "-";
  if (host.installed_version) return host.installed_version;
  if (!host.installed) return "Not installed";
  return host.installed_version || "Unknown";
}

function hostDeploymentStatusLabel(host: MunkiDeploymentHost) {
  return host.status ? deploymentStatusName(host.status) : "-";
}

function targetVersionLabel(host: MunkiDeploymentHost) {
  if (host.report_state === "current" && host.target_version) return host.target_version;
  if (host.package.strategy === "specific") return host.package.version;
  return "-";
}

function AssignedHostLastCollected({ host }: { host: MunkiDeploymentHost }) {
  if (host.report_state === "current") {
    if (!host.last_successful_at) return "Current";
    const value = <span>{formatRelative(host.last_successful_at)}</span>;
    return (
      <Tooltip>
        <TooltipTrigger render={value} />
        <TooltipContent>{formatDateTime(host.last_successful_at)}</TooltipContent>
      </Tooltip>
    );
  }

  const presentation = MUNKI_REPORT_STATES[host.report_state];
  const badge = <Badge variant={presentation.variant}>{presentation.name}</Badge>;
  return (
    <Tooltip>
      <TooltipTrigger render={badge} />
      <TooltipContent className="max-w-72 text-left">
        {host.collection_error || presentation.description}
      </TooltipContent>
    </Tooltip>
  );
}

function PackageInstallationCount({
  deployment,
}: {
  deployment: MunkiPackageDeployment | undefined;
}) {
  if (!deployment || (deployment.installed_count === 0 && deployment.assigned_count === 0)) {
    return <span className="text-muted-foreground">-</span>;
  }
  const value = (
    <span className="whitespace-nowrap tabular-nums">
      {deployment.installed_count} / {deployment.assigned_count}
    </span>
  );
  if (deployment.reporting_count === deployment.assigned_count) return value;
  return (
    <Tooltip>
      <TooltipTrigger render={value} />
      <TooltipContent>
        Current Munki data for {deployment.reporting_count} of {deployment.assigned_count}{" "}
        assignments
      </TooltipContent>
    </Tooltip>
  );
}

function MunkiSoftwareTargets({ software }: { software: MunkiSoftwareDetail }) {
  const labelsByID = useLabelNameMap();
  const packagesByID = useMemo(
    () => new Map(software.packages.map((pkg) => [pkg.id, pkg])),
    [software.packages],
  );

  return (
    <TargetDetails
      include={
        software.targets.include.length > 0 ? (
          <TableSurface variant="embedded">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Label</TableHead>
                  <TableHead>Package</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {software.targets.include.map((target) => (
                  <TableRow key={target.label_id}>
                    <TableCell>
                      <Link
                        to="/labels/$id"
                        params={{ id: String(target.label_id) }}
                        className="font-medium"
                      >
                        {labelsByID.get(target.label_id) ?? `Label ${target.label_id}`}
                      </Link>
                    </TableCell>
                    <TableCell>{munkiPackageLabel(target.package, packagesByID)}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {target.actions.map((action) => (
                          <Badge key={action} variant="secondary" className="font-normal">
                            {MUNKI_SOFTWARE_ACTIONS[action].name}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableSurface>
        ) : (
          "-"
        )
      }
      excludeLabelIDs={software.targets.exclude.map((target) => target.label_id)}
    />
  );
}

function munkiPackageLabel(
  selector: MunkiInclude["package"],
  packagesByID: ReadonlyMap<number, MunkiPackage>,
) {
  if (selector.strategy === "latest") return "Latest";
  return packagesByID.get(selector.package_id ?? 0)?.version ?? `Package ${selector.package_id}`;
}
