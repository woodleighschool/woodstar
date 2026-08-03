import { getRouteApi } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";

import { DataTable } from "@components/data-table/data-table";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { KeyValueRow, KeyValueRows } from "@components/key-value";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@components/ui/dialog";
import { useHostSoftware } from "@features/hosts/queries";
import { SoftwareIcon, softwareIconProps } from "@features/software/software-icon";
import {
  expandSoftwareSourceFilters,
  softwareSourceLabel,
  SOURCE_FILTER_OPTIONS,
  versionsSummaryLabel,
} from "@features/software/software-source-labels";
import type { HostSoftware, HostSoftwareInstalledVersion } from "@lib/api";
import { formatRelative } from "@lib/utils";

const SOURCE_FILTER_KEYS = [{ id: "source", multiple: true }] as const;
const routeApi = getRouteApi("/_authenticated/hosts/$id/software");

const softwareColumns: ColumnDef<HostSoftware>[] = [
  {
    id: "name",
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <div className="flex min-w-0 items-center gap-2">
        <SoftwareIcon {...softwareIconProps(row.original.source)} />
        <Link
          to="/software/titles/$id"
          params={{ id: String(row.original.id) }}
          className="min-w-0 truncate font-medium"
          title={row.original.name}
        >
          {row.original.name}
        </Link>
      </div>
    ),
    meta: { label: "Name" },
  },
  {
    id: "version",
    accessorFn: (row) => row.installed_versions[0].version,
    header: "Version",
    cell: ({ row }) => versionsSummaryLabel(row.original.installed_versions),
    meta: { label: "Version" },
  },
  {
    id: "source",
    accessorKey: "source",
    header: "Type",
    cell: ({ row }) => softwareSourceLabel(row.original.source, row.original.extension_for),
    meta: { label: "Type", options: SOURCE_FILTER_OPTIONS },
    enableColumnFilter: true,
  },
  {
    id: "last_opened_at",
    accessorFn: (row) => pickLatestLastOpened(row.installed_versions) ?? "",
    header: "Last Opened",
    cell: ({ row }) => {
      const lastOpenedAt = pickLatestLastOpened(row.original.installed_versions);
      return lastOpenedAt ? formatRelative(lastOpenedAt) : "-";
    },
    meta: { label: "Last Opened" },
  },
  {
    id: "path",
    header: () => "File path",
    enableSorting: false,
    cell: ({ row }) => {
      const versions = row.original.installed_versions;
      const versionLabel = versionsSummaryLabel(versions);
      const paths = installedPathsFor(versions);
      const typeLabel = softwareSourceLabel(row.original.source, row.original.extension_for);
      return (
        <InstalledPathCell
          row={row.original}
          versionLabel={versionLabel}
          typeLabel={typeLabel}
          paths={paths}
        />
      );
    },
    meta: { label: "File path" },
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
  const sources = tableSearch.filters.source ?? [];
  const query = useHostSoftware(hostId, {
    q: tableSearch.q,
    source: expandSoftwareSourceFilters(sources),
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
  });
  const data = query.data?.items ?? [];
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
        title="Failed to load software"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  if (query.isLoading) return <DataTableSkeleton columnCount={5} filterCount={1} />;
  return (
    <DataTable
      table={table}
      empty={
        <PanelEmptyState>
          {tableSearch.isFiltered ? "No matching software" : "No software yet"}
        </PanelEmptyState>
      }
    >
      <DataTableSearchInput
        value={tableSearch.q ?? ""}
        onValueChange={tableSearch.onQueryChange}
        placeholder="Search software"
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
  row,
  versionLabel,
  typeLabel,
  paths,
}: {
  row: HostSoftware;
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
      <DialogTrigger render={<Button variant="link" size="sm" className="h-auto p-0 text-xs" />}>
        {paths.length} paths
      </DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{row.name}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-4 text-sm">
          <div>
            <div className="font-medium">
              Current version{versionLabel.endsWith("versions") ? "s" : ""}:
            </div>
            <KeyValueRows className="mt-2">
              <KeyValueRow label="Version" value={versionLabel} />
              <KeyValueRow label="Type" value={typeLabel} />
            </KeyValueRows>
          </div>
          <div className="flex max-h-[60vh] flex-col gap-3 overflow-auto pr-1">
            {paths.map((item) => (
              <div key={`${item.version}-${item.path}`}>
                <div className="text-muted-foreground">Path:</div>
                <div className="break-all">{item.path}</div>
              </div>
            ))}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
function installedPathsFor(versions: HostSoftwareInstalledVersion[]): InstalledPath[] {
  return versions.flatMap((version) =>
    version.installed_paths.map((path) => ({
      path,
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
