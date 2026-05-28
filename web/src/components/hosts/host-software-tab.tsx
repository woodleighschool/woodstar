import type { ColumnDef, PaginationState, SortingState } from "@tanstack/react-table";
import { Package } from "lucide-react";
import { useMemo, useRef, useState } from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmptyState } from "@/components/data-table/data-table-empty-state";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { SoftwareIcon } from "@/components/software/software-icon";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { useHostSoftware, type HostSoftware } from "@/hooks/use-hosts";
import { tableQueryParams } from "@/hooks/use-table-pagination-params";
import type { Schemas } from "@/lib/api";
import { expandSoftwareSourceFilters, softwareSourceLabel, SOURCE_FILTER_OPTIONS } from "@/lib/software-source-labels";
import { formatRelative } from "@/lib/utils";

type InstalledVersion = Schemas["HostSoftwareInstalledVersion"];
type SignatureInfo = Schemas["PathSignatureInformation"];

const HOST_SOFTWARE_PAGE_SIZE = 50;

export function HostSoftwareTab({ hostId }: { hostId: number | null }) {
  const [draft, setDraft] = useState("");
  const [activeQuery, setActiveQuery] = useState("");
  const [sources, setSources] = useState<string[]>([]);
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: HOST_SOFTWARE_PAGE_SIZE,
  });
  const [sorting, setSorting] = useState<SortingState>([]);
  const debounceRef = useRef<number | null>(null);

  const setDraftDebounced = (next: string) => {
    setDraft(next);
    setPagination((prev) => ({ ...prev, pageIndex: 0 }));
    if (debounceRef.current !== null) {
      window.clearTimeout(debounceRef.current);
    }
    debounceRef.current = window.setTimeout(() => setActiveQuery(next.trim()), 200);
  };

  const query = useHostSoftware(hostId, {
    q: activeQuery,
    source: expandSoftwareSourceFilters(sources),
    ...tableQueryParams({ pagination, sorting }),
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = activeQuery !== "" || sources.length > 0;

  const columns = useMemo<ColumnDef<HostSoftware>[]>(
    () => [
      {
        id: "name",
        accessorFn: (row) => row.display_name || row.name,
        header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
        cell: ({ row }) => {
          const name = row.original.display_name || row.original.name;
          return (
            <span className="inline-flex items-center gap-2 truncate" title={name}>
              <SoftwareIcon source={row.original.source} />
              <span className="truncate">{name}</span>
            </span>
          );
        },
      },
      {
        id: "version",
        accessorFn: (row) => row.installed_versions?.[0]?.version ?? "",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Version" />,
        cell: ({ row }) => {
          const versions = row.original.installed_versions ?? [];
          const label =
            versions.length === 0
              ? "-"
              : versions.length === 1
                ? versions[0].version || "-"
                : `${versions.length} versions`;
          return <span className="tabular-nums">{label}</span>;
        },
      },
      {
        id: "source",
        accessorKey: "source",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
        cell: ({ row }) => (
          <span title={row.original.source}>
            {softwareSourceLabel(row.original.source, row.original.extension_for)}
          </span>
        ),
      },
      {
        id: "last_opened_at",
        accessorFn: (row) => pickLatestLastOpened(row.installed_versions ?? []) ?? "",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Last opened" />,
        cell: ({ row }) => {
          const lastOpenedAt = pickLatestLastOpened(row.original.installed_versions ?? []);
          return (
            <span title={lastOpenedAt ? new Date(lastOpenedAt).toLocaleString() : ""}>
              {lastOpenedAt ? formatRelative(lastOpenedAt) : "-"}
            </span>
          );
        },
      },
      {
        id: "path",
        header: () => "File path",
        enableSorting: false,
        cell: ({ row }) => {
          const versions = row.original.installed_versions ?? [];
          const versionLabel =
            versions.length === 0
              ? "-"
              : versions.length === 1
                ? versions[0].version || "-"
                : `${versions.length} versions`;
          const paths = installedPathsFor(versions);
          const typeLabel = softwareSourceLabel(row.original.source, row.original.extension_for);
          return (
            <InstalledPathCell row={row.original} versionLabel={versionLabel} typeLabel={typeLabel} paths={paths} />
          );
        },
        meta: { headClassName: "w-[18rem]", cellClassName: "max-w-[18rem] min-w-32" },
      },
      {
        id: "hash",
        header: () => "Hash",
        enableSorting: false,
        cell: ({ row }) => {
          const paths = installedPathsFor(row.original.installed_versions ?? []);
          return (
            <span className="font-mono text-xs" title={singleHash(paths)}>
              {truncateHash(singleHash(paths))}
            </span>
          );
        },
      },
    ],
    [],
  );

  if (query.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to load software</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
        <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
          Retry
        </Button>
      </Alert>
    );
  }

  return (
    <DataTable
      columns={columns}
      data={data}
      totalCount={totalCount}
      pagination={pagination}
      sorting={sorting}
      onPaginationChange={setPagination}
      onSortingChange={(updater) => {
        setSorting(updater);
        setPagination((prev) => ({ ...prev, pageIndex: 0 }));
      }}
      rowHref={(row) => ({ to: "/software/titles/$softwareId", params: { softwareId: String(row.id) } })}
      isLoading={query.isLoading}
      toolbar={
        <div className="flex items-center gap-2">
          <DataTableSearch
            value={draft}
            onChange={(next) => {
              if (next === "") {
                setDraft("");
                setActiveQuery("");
                setPagination((prev) => ({ ...prev, pageIndex: 0 }));
                return;
              }
              setDraftDebounced(next);
            }}
            placeholder="Search software"
            label="Search host software"
          />
          <DataTableFacetedFilter
            title="Type"
            options={SOURCE_FILTER_OPTIONS}
            selected={sources}
            onChange={(next) => {
              setSources(next);
              setPagination((prev) => ({ ...prev, pageIndex: 0 }));
            }}
          />
        </div>
      }
      empty={
        <DataTableEmptyState
          icon={<Package />}
          title={hasFilters ? "No matches" : "No software inventory yet"}
          description={
            hasFilters
              ? "No software matched the current filters."
              : "osquery will populate this on next detail refresh."
          }
        />
      }
    />
  );
}

interface InstalledPath {
  path: string;
  version: string;
  signature?: SignatureInfo;
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
    return <span>-</span>;
  }
  if (paths.length === 1) {
    return (
      <span className="block truncate" title={paths[0].path}>
        {paths[0].path}
      </span>
    );
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

function installedPathsFor(versions: InstalledVersion[]): InstalledPath[] {
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

function pickLatestLastOpened(versions: InstalledVersion[]): string | undefined {
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

function buildSignatureIndex(rows: SignatureInfo[] | null | undefined): Map<string, SignatureInfo> {
  const map = new Map<string, SignatureInfo>();
  if (!rows) return map;
  for (const row of rows) {
    if (row.installed_path) map.set(row.installed_path, row);
  }
  return map;
}
