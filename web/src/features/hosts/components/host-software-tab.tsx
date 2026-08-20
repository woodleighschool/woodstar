import { getRouteApi } from "@tanstack/react-router";
import { useMemo } from "react";

import { DataTable } from "@components/data-table/data-table";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import type { DataTableColumnDef } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { TextLink } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { useHostSoftware } from "@features/hosts/queries";
import { SoftwareIcon, softwareIconProps } from "@features/software/software-icon";
import {
  expandSoftwareSourceFilters,
  softwareSourceLabel,
  SOURCE_FILTER_OPTIONS,
  versionsSummaryLabel,
} from "@features/software/software-source-labels";
import type { HostSoftware, HostSoftwareInstalledVersion } from "@lib/api";
import { countLabel, formatRelative } from "@lib/utils";

const SOURCE_FILTER_KEYS = [{ id: "source", multiple: true }] as const;
const routeApi = getRouteApi("/_authenticated/hosts/$id/software");

const softwareColumns: DataTableColumnDef<HostSoftware>[] = [
  {
    id: "name",
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <div className="flex min-w-0 items-center gap-2">
        <SoftwareIcon {...softwareIconProps(row.original.source)} />
        <TextLink
          to="/software/titles/$id"
          params={{ id: String(row.original.id) }}
          className="min-w-0 truncate font-medium"
          title={row.original.name}
        >
          {row.original.name}
        </TextLink>
      </div>
    ),
    size: 240,
    minSize: 160,
    meta: { label: "Name" },
  },
  {
    id: "version",
    accessorFn: (row) => row.installed_versions[0].version,
    header: "Version",
    cell: ({ row }) => versionsSummaryLabel(row.original.installed_versions),
    size: 112,
    minSize: 112,
    maxSize: 112,
    enableResizing: false,
    meta: { label: "Version" },
  },
  {
    id: "source",
    accessorKey: "source",
    header: "Type",
    cell: ({ row }) => softwareSourceLabel(row.original.source, row.original.extension_for),
    size: 144,
    minSize: 144,
    maxSize: 144,
    enableResizing: false,
    meta: { label: "Type", options: SOURCE_FILTER_OPTIONS },
    enableColumnFilter: true,
    filterFn: () => true,
  },
  {
    id: "last_opened_at",
    accessorFn: (row) => pickLatestLastOpened(row.installed_versions) ?? "",
    header: "Last Opened",
    cell: ({ row }) => {
      const lastOpenedAt = pickLatestLastOpened(row.original.installed_versions);
      return lastOpenedAt ? formatRelative(lastOpenedAt) : "-";
    },
    size: 136,
    minSize: 136,
    maxSize: 136,
    enableResizing: false,
    meta: { label: "Last Opened" },
  },
  {
    id: "path",
    header: () => "Installed Path",
    enableSorting: false,
    cell: ({ row }) => {
      const versions = row.original.installed_versions;
      const versionLabel = versionsSummaryLabel(versions);
      const paths = installedPathsFor(versions);
      const typeLabel = softwareSourceLabel(row.original.source, row.original.extension_for);
      return (
        <InstalledPathCell
          software={row.original}
          versionLabel={versionLabel}
          typeLabel={typeLabel}
          paths={paths}
        />
      );
    },
    size: 360,
    minSize: 240,
    meta: { label: "Installed Path" },
  },
];
export function HostSoftwareTab({ hostId }: { hostId: number | null }) {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: SOURCE_FILTER_KEYS,
  });
  const sources = search.source ?? [];
  const query = useHostSoftware(hostId, {
    q: tableSearch.q,
    source: expandSoftwareSourceFilters(sources),
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });
  const data = useMemo(() => query.data?.items ?? [], [query.data?.items]);
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data,
    columns: softwareColumns,
    getRowId: (row) => String(row.id),
    pageCount,
    rowCount: totalCount,
  });
  if (query.error) {
    return (
      <QueryError
        title="Failed to Load Software"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  if (query.isLoading) return <DataTableSkeleton columnCount={5} filterCount={1} />;
  return (
    <DataTable
      table={table}
      pending={query.isPlaceholderData}
      empty={
        <PanelEmptyState>
          {tableSearch.isFiltered ? "No Matching Software" : "No Observed Software"}
        </PanelEmptyState>
      }
    >
      <DataTableSearchInput
        loading={query.isPlaceholderData}
        value={tableSearch.q ?? ""}
        onValueChange={tableSearch.onQueryChange}
        placeholder="Search Software"
      />
      <DataTableFacetedFilter
        column={table.getColumn("source")}
        title="Type"
        options={SOURCE_FILTER_OPTIONS}
      />
    </DataTable>
  );
}

interface InstalledPath {
  path: string;
  version: string;
}
function InstalledPathCell({
  software,
  versionLabel,
  typeLabel,
  paths,
}: {
  software: HostSoftware;
  versionLabel: string;
  typeLabel: string;
  paths: InstalledPath[];
}) {
  if (paths.length === 0) {
    return "-";
  }
  if (paths.length === 1) {
    return paths[0].path;
  }
  return (
    <Dialog>
      <DialogTrigger render={<Button variant="link" size="sm" />}>
        {countLabel(paths.length, "path")}
      </DialogTrigger>
      <DialogContent className="max-h-[85vh] gap-0 overflow-hidden p-0 sm:max-w-4xl">
        <DialogHeader className="border-b px-4 py-3 pr-12">
          <DialogTitle>{software.name}</DialogTitle>
          <DialogDescription>
            {versionLabel}, {typeLabel}, {countLabel(paths.length, "installed path")}
          </DialogDescription>
        </DialogHeader>
        <div className="min-h-0 overflow-y-auto p-4">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Version</TableHead>
                <TableHead>Installed Path</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paths.map((item) => (
                <TableRow key={`${item.version}-${item.path}`}>
                  <TableCell className="align-top">{item.version}</TableCell>
                  <TableCell>{item.path}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </DialogContent>
    </Dialog>
  );
}
function installedPathsFor(versions: HostSoftwareInstalledVersion[]): InstalledPath[] {
  return versions.flatMap((version) =>
    version.paths.map((item) => ({
      path: item.path,
      version: version.version,
    })),
  );
}
function pickLatestLastOpened(versions: HostSoftwareInstalledVersion[]): string | undefined {
  let latest: string | undefined;
  for (const version of versions) {
    const value = version.last_opened_at;
    if (!value) continue;
    const time = new Date(value).getTime();
    if (Number.isNaN(time)) continue;
    if (!latest || time > new Date(latest).getTime()) latest = value;
  }
  return latest;
}
