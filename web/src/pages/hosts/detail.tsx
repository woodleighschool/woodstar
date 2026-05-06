import { Link, useParams } from "@tanstack/react-router";
import { ChevronLeft, Package, Search, X } from "lucide-react";
import type React from "react";
import { useEffect, useRef, useState } from "react";

import { EmptyState } from "@/components/feedback/empty-state";
import { ErrorState } from "@/components/feedback/error-state";
import { Spinner } from "@/components/feedback/spinner";
import { PageHeader } from "@/components/layout/page-header";
import { FilterPopover, type FilterGroup } from "@/components/lists/filter-popover";
import type { SortState } from "@/components/lists/sort-state";
import { SortableTableHead } from "@/components/lists/sortable-table-head";
import { TablePagination } from "@/components/lists/table-pagination";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useHost, useHostSoftware, type Host, type HostSoftware } from "@/hooks/use-hosts";
import type { Schemas } from "@/lib/api";
import { deviceMappingSourceLabel } from "@/lib/device-mapping-source-labels";
import { softwareSourceLabel, SOURCE_FILTER_OPTIONS } from "@/lib/software-source-labels";
import { formatBytes, formatRelative } from "@/lib/utils";

type InstalledVersion = Schemas["HostSoftwareInstalledVersionBody"];
type SignatureInfo = Schemas["PathSignatureInformationBody"];
const HOST_SOFTWARE_PAGE_SIZE = 50;
const SEARCH_DEBOUNCE_MS = 200;

export function HostDetailPage() {
  const { hostId } = useParams({ from: "/_authed/hosts/$hostId" });
  const query = useHost(hostId);
  const host = query.data;

  return (
    <div className="flex flex-col">
      <PageHeader
        title={host?.display_name || `Host ${hostId}`}
        description={host?.hardware_uuid}
        actions={
          <Button asChild variant="outline" size="sm">
            <Link to="/hosts" className="gap-1">
              <ChevronLeft className="size-4" /> Back to hosts
            </Link>
          </Button>
        }
      />

      <div className="p-6">
        {query.error ? (
          <ErrorState message={query.error.message} onRetry={() => query.refetch()} />
        ) : query.isLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Spinner /> Loading…
          </div>
        ) : !host ? null : (
          <Tabs defaultValue="details">
            <TabsList>
              <TabsTrigger value="details">Details</TabsTrigger>
              <TabsTrigger value="software">Software</TabsTrigger>
            </TabsList>

            <TabsContent value="details">
              <DetailsTab host={host} />
            </TabsContent>

            <TabsContent value="software">
              <SoftwareTab hostId={hostId} />
            </TabsContent>
          </Tabs>
        )}
      </div>
    </div>
  );
}

function DetailsTab({ host }: { host: Host }) {
  const primaryMapping = host.device_mappings?.[0];
  const primaryUser = primaryMapping
    ? `${primaryMapping.email} (${deviceMappingSourceLabel(primaryMapping.source)})`
    : "-";

  return (
    <DefinitionList
      rows={[
        ["Hardware UUID", <span className="font-mono text-xs">{host.hardware_uuid}</span>],
        ["Vendor", host.hardware_vendor || "-"],
        ["Model", host.hardware_model || "-"],
        ["CPU", host.cpu_brand ? `${host.cpu_brand} (${host.cpu_logical_cores} logical cores)` : "-"],
        ["Memory", host.physical_memory > 0 ? formatBytes(host.physical_memory) : "-"],
        ["Hostname", host.hostname || "-"],
        ["Computer name", host.computer_name || "-"],
        ["Platform", host.platform ? <Badge variant="muted">{host.platform}</Badge> : "-"],
        ["OS version", host.os_version || "-"],
        ["Kernel", host.kernel_version || "-"],
        ["Serial", host.hardware_serial || "-"],
        ["osquery version", host.osquery_version || "-"],
        ["Orbit version", host.orbit_version || "-"],
        ["Primary user", primaryUser],
        [
          "Enrolled",
          host.enrolled_at ? (
            <span title={new Date(host.enrolled_at).toLocaleString()}>{formatRelative(host.enrolled_at)}</span>
          ) : (
            "-"
          ),
        ],
        [
          "Last seen",
          host.last_seen_at ? (
            <span title={new Date(host.last_seen_at).toLocaleString()}>{formatRelative(host.last_seen_at)}</span>
          ) : (
            "-"
          ),
        ],
        [
          "Detail refreshed",
          host.detail_updated_at ? (
            <span title={new Date(host.detail_updated_at).toLocaleString()}>
              {formatRelative(host.detail_updated_at)}
            </span>
          ) : (
            <span className="text-muted-foreground">never</span>
          ),
        ],
      ]}
    />
  );
}

function SoftwareTab({ hostId }: { hostId: string }) {
  const [searchInput, setSearchInput] = useState("");
  const [activeQuery, setActiveQuery] = useState("");
  const [sourceFilters, setSourceFilters] = useState<string[]>([]);
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(HOST_SOFTWARE_PAGE_SIZE);
  const [sort, setSort] = useState<SortState>({});
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const query = useHostSoftware(hostId, {
    q: activeQuery,
    source: sourceFilters,
    page: page - 1,
    per_page: perPage,
    order_key: sort.orderKey,
    order_direction: sort.orderDirection,
  });

  useEffect(
    () => () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    },
    [],
  );

  if (query.error) {
    return <ErrorState message={query.error.message} onRetry={() => query.refetch()} />;
  }

  if (query.isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Spinner /> Loading…
      </div>
    );
  }

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? data.length;
  const hasFilters = activeQuery !== "" || sourceFilters.length > 0;
  const filterGroups: FilterGroup[] = [
    {
      id: "type",
      label: "Type",
      options: SOURCE_FILTER_OPTIONS,
      selected: sourceFilters,
      onChange: (next) => {
        setSourceFilters(next);
        setPage(1);
      },
    },
  ];

  if (data.length === 0 && !hasFilters) {
    return (
      <div className="rounded-md border border-dashed bg-muted/30 px-4 py-6 text-sm">
        <p className="font-medium">No software inventory yet</p>
        <p className="text-muted-foreground">osquery will populate this on next detail refresh.</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-md">
          <Search
            className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden
          />
          <Input
            value={searchInput}
            onChange={(event) => {
              setSearchInput(event.target.value);
              setPage(1);
              if (debounceRef.current) clearTimeout(debounceRef.current);
              debounceRef.current = setTimeout(() => setActiveQuery(event.target.value.trim()), SEARCH_DEBOUNCE_MS);
            }}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                if (debounceRef.current) clearTimeout(debounceRef.current);
                setActiveQuery(searchInput.trim());
              }
            }}
            placeholder="Search software"
            className="pl-8 pr-8"
            aria-label="Search host software"
          />
          {searchInput ? (
            <button
              type="button"
              onClick={() => {
                if (debounceRef.current) clearTimeout(debounceRef.current);
                setSearchInput("");
                setActiveQuery("");
                setPage(1);
              }}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-muted-foreground hover:text-foreground"
              aria-label="Clear search"
            >
              <X className="size-3.5" />
            </button>
          ) : null}
        </div>
        <div className="text-xs text-muted-foreground tabular-nums">
          {query.isFetching ? (
            <span className="inline-flex items-center gap-1">
              <Spinner className="size-3" /> Loading…
            </span>
          ) : (
            <>
              {totalCount} app{totalCount === 1 ? "" : "s"}
            </>
          )}
        </div>
        <FilterPopover groups={filterGroups} />
      </div>

      {data.length === 0 ? (
        <EmptyState
          icon={Package}
          title="No matches"
          description={hasFilters ? "No software matched the current filters." : "No software inventory yet."}
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <SortableTableHead
                  orderKey="name"
                  active={sort}
                  onSort={(next) => {
                    setSort(next);
                    setPage(1);
                  }}
                >
                  Name
                </SortableTableHead>
                <SortableTableHead
                  orderKey="version"
                  active={sort}
                  onSort={(next) => {
                    setSort(next);
                    setPage(1);
                  }}
                >
                  Installed version
                </SortableTableHead>
                <SortableTableHead
                  orderKey="source"
                  active={sort}
                  onSort={(next) => {
                    setSort(next);
                    setPage(1);
                  }}
                >
                  Type
                </SortableTableHead>
                <SortableTableHead
                  orderKey="last_opened_at"
                  active={sort}
                  onSort={(next) => {
                    setSort(next);
                    setPage(1);
                  }}
                >
                  Last opened
                </SortableTableHead>
                <TableHead>File path</TableHead>
                <TableHead>Hash</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((row) => (
                <HostSoftwareRow key={row.id} row={row} />
              ))}
            </TableBody>
          </Table>
          <TablePagination
            page={page}
            perPage={perPage}
            totalCount={totalCount}
            visibleCount={data.length}
            onPageChange={setPage}
            onPerPageChange={(next) => {
              setPerPage(next);
              setPage(1);
            }}
          />
        </div>
      )}
    </div>
  );
}

function HostSoftwareRow({ row }: { row: HostSoftware }) {
  const versions = row.installed_versions ?? [];
  const versionLabel =
    versions.length === 0 ? "-" : versions.length === 1 ? versions[0].version || "-" : `${versions.length} versions`;
  const lastOpenedAt = pickLatestLastOpened(versions);
  const displayName = row.display_name || row.name;
  const paths = installedPathsFor(versions);
  const typeLabel = softwareSourceLabel(row.source, row.extension_for);

  return (
    <TableRow>
      <TableCell className="font-medium">
        <div className="min-w-0">
          <Link
            to="/software/titles/$softwareId"
            params={{ softwareId: row.id }}
            className="block truncate hover:underline"
            title={displayName}
          >
            {displayName}
          </Link>
        </div>
      </TableCell>
      <TableCell className="text-muted-foreground tabular-nums">{versionLabel}</TableCell>
      <TableCell className="text-muted-foreground" title={row.source}>
        {typeLabel}
      </TableCell>
      <TableCell className="text-muted-foreground" title={lastOpenedAt ? new Date(lastOpenedAt).toLocaleString() : ""}>
        {lastOpenedAt ? formatRelative(lastOpenedAt) : "-"}
      </TableCell>
      <TableCell className="max-w-[28rem]">
        <InstalledPathCell row={row} versionLabel={versionLabel} typeLabel={typeLabel} paths={paths} />
      </TableCell>
      <TableCell className="font-mono text-xs text-muted-foreground">{singleHash(paths)}</TableCell>
    </TableRow>
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
    return <span className="text-muted-foreground">-</span>;
  }
  if (paths.length === 1) {
    return (
      <span className="font-mono text-xs text-muted-foreground break-all" title={paths[0].path}>
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
              <dd className="font-mono text-xs">{versionLabel}</dd>
              <dt className="text-muted-foreground">Type</dt>
              <dd>{typeLabel}</dd>
            </dl>
          </div>
          <div className="flex max-h-[60vh] flex-col gap-3 overflow-auto pr-1">
            {paths.map((item) => (
              <div key={`${item.version}-${item.path}`}>
                <div className="text-muted-foreground">Path:</div>
                <div className="font-mono text-xs break-all">{item.path}</div>
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
  return paths[0].signature?.hash_sha256 || "-";
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

function DefinitionList({ rows }: { rows: Array<[string, React.ReactNode]> }) {
  return (
    <dl className="rounded-lg border bg-card divide-y">
      {rows.map(([label, value]) => (
        <div key={label} className="grid grid-cols-[10rem_1fr] gap-3 px-4 py-2 text-sm">
          <dt className="text-muted-foreground">{label}</dt>
          <dd className="font-medium break-all">{value ?? "-"}</dd>
        </div>
      ))}
    </dl>
  );
}
