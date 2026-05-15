import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, Package } from "lucide-react";
import { useRef, useState } from "react";

import { CheckStatusBadge } from "@/components/checks/check-status-badge";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import {
  HostCertificatesCard,
  HostInfoCard,
  HostLabelsCard,
  HostUsersCard,
} from "@/components/hosts/host-detail-cards";
import { HostHeader } from "@/components/hosts/host-header";
import { SoftwareIcon } from "@/components/software/software-icon";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { PageTabs, PageTabsContent, PageTabsList, PageTabsTrigger } from "@/components/ui/page-tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
  useHost,
  useHostChecks,
  useHostQueries,
  useHostSoftware,
  type HostReport,
  type HostSoftware,
} from "@/hooks/use-hosts";
import type { Schemas } from "@/lib/api";
import { softwareSourceLabel, SOURCE_FILTER_OPTIONS } from "@/lib/software-source-labels";
import { formatRelative } from "@/lib/utils";

type InstalledVersion = Schemas["HostSoftwareInstalledVersion"];
type SignatureInfo = Schemas["PathSignatureInformation"];

const HOST_SOFTWARE_PAGE_SIZE = 50;

export function HostDetailPage() {
  const { hostId } = useParams({ from: "/_authenticated/hosts/$hostId" });
  const query = useHost(hostId);
  const host = query.data;

  if (query.error) {
    return (
      <div className="p-6">
        <Alert variant="destructive">
          <AlertTitle>Failed to load host</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
          <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
            Retry
          </Button>
        </Alert>
      </div>
    );
  }

  if (query.isLoading || !host) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading...
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      <HostHeader host={host} />

      <PageTabs defaultValue="details">
        <PageTabsList>
          <PageTabsTrigger value="details">Details</PageTabsTrigger>
          <PageTabsTrigger value="software">Software</PageTabsTrigger>
          <PageTabsTrigger value="reports">Reports</PageTabsTrigger>
          <PageTabsTrigger value="checks">Checks</PageTabsTrigger>
        </PageTabsList>

        <PageTabsContent value="details">
          <div className="flex flex-col gap-4">
            <HostInfoCard host={host} />
            <div className="grid grid-cols-1 items-start gap-4 lg:grid-cols-2">
              <HostLabelsCard host={host} />
              <HostUsersCard host={host} />
            </div>
            <HostCertificatesCard host={host} />
          </div>
        </PageTabsContent>

        <PageTabsContent value="software">
          <SoftwareTab hostId={hostId} />
        </PageTabsContent>

        <PageTabsContent value="reports">
          <HostReportsTab hostId={hostId} />
        </PageTabsContent>

        <PageTabsContent value="checks">
          <HostChecksTab hostId={hostId} />
        </PageTabsContent>
      </PageTabs>
    </div>
  );
}

function HostReportsTab({ hostId }: { hostId: string }) {
  const query = useHostQueries(hostId);
  const rows = query.data?.items ?? [];
  if (query.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading reports...
      </div>
    );
  }
  if (rows.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyTitle>No reports</EmptyTitle>
          <EmptyDescription>Add a scheduled report to view custom vitals for this host.</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }
  return (
    <div className="grid gap-3">
      {rows.map((report) => (
        <HostReportCard key={report.report_id} hostId={hostId} report={report} />
      ))}
    </div>
  );
}

function HostReportCard({ hostId, report }: { hostId: string; report: HostReport }) {
  const entries = Object.entries(report.first_result ?? {}).sort(([a], [b]) => a.localeCompare(b));
  return (
    <div className="rounded-lg border p-4">
      <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <Link
            to="/hosts/$hostId/reports/$reportId"
            params={{ hostId, reportId: String(report.report_id) }}
            className="font-medium hover:underline"
          >
            {report.name}
          </Link>
          <p className="text-muted-foreground mt-1 text-xs">
            {report.last_fetched ? `Updated ${formatRelative(report.last_fetched)}` : "Collecting results"}
            {report.n_host_results > 0
              ? ` · ${report.n_host_results} row${report.n_host_results === 1 ? "" : "s"}`
              : ""}
          </p>
        </div>
        <Button asChild size="sm" variant="outline">
          <Link to="/reports/$reportId" params={{ reportId: String(report.report_id) }}>
            All hosts
          </Link>
        </Button>
      </div>
      {entries.length > 0 ? (
        <dl className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5">
          {entries.map(([key, value]) => (
            <div key={key} className="flex min-w-0 flex-col gap-1">
              <dt className="text-muted-foreground text-xs font-semibold">{key}</dt>
              <dd className="text-foreground truncate text-sm" title={value}>
                {value || "-"}
              </dd>
            </div>
          ))}
        </dl>
      ) : (
        <p className="text-muted-foreground text-sm">
          {report.last_fetched
            ? "This report ran but returned no rows for this host."
            : "Waiting for this host to run the report."}
        </p>
      )}
    </div>
  );
}

function HostChecksTab({ hostId }: { hostId: string }) {
  const query = useHostChecks(hostId);
  const rows = query.data?.items ?? [];
  return (
    <div className="rounded-lg border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Check</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Last evaluated</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow key={`${row.check_id}-${row.host_id}`}>
              <TableCell>
                <Link
                  to="/checks/$checkId"
                  params={{ checkId: String(row.check_id) }}
                  className="font-medium hover:underline"
                >
                  {row.check_name || String(row.check_id)}
                </Link>
              </TableCell>
              <TableCell>
                <CheckStatusBadge response={row.response} />
              </TableCell>
              <TableCell className="text-muted-foreground">
                {row.updated_at ? formatRelative(row.updated_at) : "-"}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function SoftwareTab({ hostId }: { hostId: string }) {
  const [draft, setDraft] = useState("");
  const [activeQuery, setActiveQuery] = useState("");
  const [sources, setSources] = useState<string[]>([]);
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(HOST_SOFTWARE_PAGE_SIZE);
  const [orderKey, setOrderKey] = useState<string | undefined>(undefined);
  const [orderDirection, setOrderDirection] = useState<"asc" | "desc" | undefined>(undefined);
  const debounceRef = useRef<number | null>(null);

  const setDraftDebounced = (next: string) => {
    setDraft(next);
    setPage(1);
    if (debounceRef.current !== null) {
      window.clearTimeout(debounceRef.current);
    }
    debounceRef.current = window.setTimeout(() => setActiveQuery(next.trim()), 200);
  };

  const query = useHostSoftware(hostId, {
    q: activeQuery,
    source: sources,
    page,
    per_page: perPage,
    order_key: orderKey,
    order_direction: orderDirection,
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = activeQuery !== "" || sources.length > 0;

  const columns: ColumnDef<HostSoftware>[] = [
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
        return <span className="text-muted-foreground tabular-nums">{label}</span>;
      },
    },
    {
      id: "source",
      accessorKey: "source",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
      cell: ({ row }) => (
        <span className="text-muted-foreground" title={row.original.source}>
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
          <span className="text-muted-foreground" title={lastOpenedAt ? new Date(lastOpenedAt).toLocaleString() : ""}>
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
        return <InstalledPathCell row={row.original} versionLabel={versionLabel} typeLabel={typeLabel} paths={paths} />;
      },
      meta: { cellClassName: "max-w-[28rem] whitespace-normal break-all" },
    },
    {
      id: "hash",
      header: () => "Hash",
      enableSorting: false,
      cell: ({ row }) => {
        const paths = installedPathsFor(row.original.installed_versions ?? []);
        return (
          <span className="text-muted-foreground font-mono text-xs" title={singleHash(paths)}>
            {truncateHash(singleHash(paths))}
          </span>
        );
      },
    },
  ];

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
      page={page}
      perPage={perPage}
      sort={{ orderKey, orderDirection }}
      rowHref={(row) => ({ to: "/software/titles/$softwareId", params: { softwareId: String(row.id) } })}
      onPageChange={setPage}
      onPerPageChange={(n) => {
        setPerPage(n);
        setPage(1);
      }}
      onSortChange={(s) => {
        setOrderKey(s.orderKey);
        setOrderDirection(s.orderDirection);
        setPage(1);
      }}
      isLoading={query.isLoading}
      toolbar={
        <div className="flex items-center gap-2">
          <DataTableSearch
            value={draft}
            onChange={(next) => {
              if (next === "") {
                setDraft("");
                setActiveQuery("");
                setPage(1);
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
              setPage(1);
            }}
          />
          <div className="text-muted-foreground ml-auto text-xs tabular-nums">
            {query.isFetching ? "Loading..." : `${totalCount} ${totalCount === 1 ? "app" : "apps"}`}
          </div>
        </div>
      }
      empty={
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Package />
            </EmptyMedia>
            <EmptyTitle>{hasFilters ? "No matches" : "No software inventory yet"}</EmptyTitle>
            <EmptyDescription>
              {hasFilters
                ? "No software matched the current filters."
                : "osquery will populate this on next detail refresh."}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
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
    return <span className="text-muted-foreground">-</span>;
  }
  if (paths.length === 1) {
    return (
      <span className="text-muted-foreground" title={paths[0].path}>
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
