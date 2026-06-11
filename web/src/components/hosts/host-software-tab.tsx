import { Link } from "@tanstack/react-router";
import {
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
  type ColumnFiltersState,
  type PaginationState,
  type SortingState,
} from "@tanstack/react-table";
import { useRef, useState } from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { EmptyPanel } from "@/components/empty-panel";
import { QueryError } from "@/components/query-error";
import { SoftwareIcon } from "@/components/software/software-icon";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { encodeSort } from "@/hooks/use-data-table-search";
import { useHostSoftware, type HostSoftware } from "@/hooks/use-hosts";
import type { HostSoftwareInstalledVersion, PathSignatureInformation } from "@/lib/api";
import { formatRelative } from "@/lib/utils";
import {
  expandSoftwareSourceFilters,
  softwareSourceLabel,
  SOURCE_FILTER_OPTIONS,
  versionsSummaryLabel,
} from "@/pages/software/software-source-labels";

const HOST_SOFTWARE_PAGE_SIZE = 50;

const softwareColumns: ColumnDef<HostSoftware>[] = [
  {
    id: "name",
    accessorFn: (row) => row.display_name || row.name,
    header: ({ column }) => <DataTableColumnHeader column={column} label="Name" />,
    cell: ({ row }) => (
      <Link
        to="/software/titles/$softwareId"
        params={{ softwareId: String(row.original.id) }}
        className="inline-flex items-center gap-2 truncate font-medium hover:underline"
      >
        <SoftwareIcon source={row.original.source} />
        <span className="truncate">{row.original.display_name || row.original.name}</span>
      </Link>
    ),
    meta: { label: "Name" },
  },
  {
    id: "version",
    accessorFn: (row) => row.installed_versions?.[0]?.version ?? "",
    header: ({ column }) => <DataTableColumnHeader column={column} label="Version" />,
    cell: ({ row }) => versionsSummaryLabel(row.original.installed_versions ?? []),
    meta: { label: "Version" },
  },
  {
    id: "source",
    accessorKey: "source",
    header: ({ column }) => <DataTableColumnHeader column={column} label="Type" />,
    cell: ({ row }) => softwareSourceLabel(row.original.source, row.original.extension_for),
    meta: { label: "Type", variant: "multiSelect", options: SOURCE_FILTER_OPTIONS },
    enableColumnFilter: true,
  },
  {
    id: "last_opened_at",
    accessorFn: (row) => pickLatestLastOpened(row.installed_versions ?? []) ?? "",
    header: ({ column }) => <DataTableColumnHeader column={column} label="Last Opened" />,
    cell: ({ row }) => {
      const lastOpenedAt = pickLatestLastOpened(row.original.installed_versions ?? []);
      return lastOpenedAt ? formatRelative(lastOpenedAt) : "-";
    },
    meta: { label: "Last Opened" },
  },
  {
    id: "path",
    header: () => "File path",
    enableSorting: false,
    cell: ({ row }) => {
      const versions = row.original.installed_versions ?? [];
      const versionLabel = versionsSummaryLabel(versions);
      const paths = installedPathsFor(versions);
      const typeLabel = softwareSourceLabel(row.original.source, row.original.extension_for);
      return <InstalledPathCell row={row.original} versionLabel={versionLabel} typeLabel={typeLabel} paths={paths} />;
    },
    meta: { label: "File path" },
  },
  {
    id: "hash",
    header: () => "Hash",
    enableSorting: false,
    cell: ({ row }) => truncateHash(singleHash(installedPathsFor(row.original.installed_versions ?? []))),
    meta: { label: "Hash" },
  },
];

export function HostSoftwareTab({ hostId }: { hostId: number | null }) {
  const [draft, setDraft] = useState("");
  const [activeQuery, setActiveQuery] = useState("");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: HOST_SOFTWARE_PAGE_SIZE });
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const debounceRef = useRef<number | null>(null);

  const sources = (columnFilters.find((filter) => filter.id === "source")?.value as string[] | undefined) ?? [];

  const setSearch = (next: string) => {
    setDraft(next);
    setPagination((prev) => ({ ...prev, pageIndex: 0 }));
    if (debounceRef.current !== null) window.clearTimeout(debounceRef.current);
    if (next === "") {
      setActiveQuery("");
      return;
    }
    debounceRef.current = window.setTimeout(() => setActiveQuery(next.trim()), 200);
  };

  const query = useHostSoftware(hostId, {
    q: activeQuery,
    source: expandSoftwareSourceFilters(sources),
    page: pagination.pageIndex + 1,
    per_page: pagination.pageSize,
    sort: sorting.length > 0 ? encodeSort(sorting[0].id, sorting[0].desc) : undefined,
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = activeQuery !== "" || sources.length > 0;

  const table = useReactTable({
    data,
    columns: softwareColumns,
    getCoreRowModel: getCoreRowModel(),
    getRowId: (row) => String(row.id),
    manualPagination: true,
    manualSorting: true,
    manualFiltering: true,
    pageCount: Math.max(1, Math.ceil(totalCount / pagination.pageSize)),
    state: { pagination, sorting, columnFilters },
    onPaginationChange: setPagination,
    onSortingChange: (updater) => {
      setSorting(updater);
      setPagination((prev) => ({ ...prev, pageIndex: 0 }));
    },
    onColumnFiltersChange: (updater) => {
      setColumnFilters(updater);
      setPagination((prev) => ({ ...prev, pageIndex: 0 }));
    },
  });

  if (query.error) {
    return <QueryError title="Failed to load software" error={query.error} onRetry={() => void query.refetch()} />;
  }
  if (query.isLoading) return <DataTableSkeleton columnCount={6} filterCount={1} />;

  return (
    <DataTable table={table} empty={<EmptyPanel>{hasFilters ? "No matching software" : "No software yet"}</EmptyPanel>}>
      <div className="flex flex-wrap items-center gap-2 p-1">
        <Input
          value={draft}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="Search software"
          className="h-8 w-56"
        />
        <DataTableFacetedFilter
          column={table.getColumn("source")}
          title="Type"
          options={SOURCE_FILTER_OPTIONS}
          multiple
        />
      </div>
    </DataTable>
  );
}

interface InstalledPath {
  path: string;
  version: string;
  signature?: PathSignatureInformation;
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
      <DialogTrigger asChild>
        <Button variant="link" size="sm" className="h-auto px-0 py-0 text-xs">
          {paths.length} paths
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{row.display_name || row.name}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-4 text-sm">
          <div>
            <div className="font-medium">Current version{versionLabel.endsWith("versions") ? "s" : ""}:</div>
            <dl className="mt-2 grid grid-cols-[7rem_1fr] gap-x-3 gap-y-1">
              <dt className="text-muted-foreground">Version</dt>
              <dd>{versionLabel}</dd>
              <dt className="text-muted-foreground">Type</dt>
              <dd>{typeLabel}</dd>
            </dl>
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
  return versions.flatMap((version) => {
    const signatures = buildSignatureIndex(version.signature_information);
    return (version.installed_paths ?? []).map((path) => ({
      path,
      version: version.version || "",
      signature: signatures.get(path),
    }));
  });
}

function singleHash(paths: InstalledPath[]): string {
  if (paths.length !== 1) return "-";
  return paths[0].signature?.hash_sha256 ?? "-";
}

function truncateHash(hash: string): string {
  if (hash === "-" || hash.length <= 16) return hash;
  return `${hash.slice(0, 8)}…${hash.slice(-8)}`;
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

function buildSignatureIndex(
  rows: PathSignatureInformation[] | null | undefined,
): Map<string, PathSignatureInformation> {
  const map = new Map<string, PathSignatureInformation>();
  if (!rows) return map;
  for (const row of rows) {
    if (row.installed_path) map.set(row.installed_path, row);
  }
  return map;
}
