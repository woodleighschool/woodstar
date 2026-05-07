import { Link, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, Package, Search, X } from "lucide-react";
import { useState } from "react";

import { HostInfoCard, HostLabelsCard, HostUsersCard } from "@/components/hosts/host-detail-cards";
import { HostHeader } from "@/components/hosts/host-header";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/ui/data-table";
import { DataTableColumnHeader } from "@/components/ui/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/ui/data-table-faceted-filter";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { PageTabs, PageTabsContent, PageTabsList, PageTabsTrigger } from "@/components/ui/page-tabs";
import { useHost, useHostSoftware, type HostSoftware } from "@/hooks/use-hosts";
import type { Schemas } from "@/lib/api";
import { softwareSourceLabel, SOURCE_FILTER_OPTIONS } from "@/lib/software-source-labels";
import { formatRelative } from "@/lib/utils";

type InstalledVersion = Schemas["HostSoftwareInstalledVersionBody"];
type SignatureInfo = Schemas["PathSignatureInformationBody"];

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
          <Button variant="outline" size="sm" onClick={() => query.refetch()} className="mt-2 w-fit">
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
        </PageTabsList>

        <PageTabsContent value="details">
          <HostInfoCard host={host} />
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <HostLabelsCard host={host} />
            <HostUsersCard host={host} />
          </div>
        </PageTabsContent>

        <PageTabsContent value="software">
          <SoftwareTab hostId={hostId} />
        </PageTabsContent>
      </PageTabs>
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

  const setDraftDebounced = (next: string) => {
    setDraft(next);
    setPage(1);
    window.clearTimeout(debounceTimer);
    debounceTimer = window.setTimeout(() => setActiveQuery(next.trim()), 200);
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
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => {
        const name = row.original.display_name || row.original.name;
        return (
          <Link
            to="/software/titles/$softwareId"
            params={{ softwareId: row.original.id }}
            className="block truncate hover:underline"
            title={name}
          >
            {name}
          </Link>
        );
      },
    },
    {
      id: "version",
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
      header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
      cell: ({ row }) => (
        <span className="text-muted-foreground" title={row.original.source}>
          {softwareSourceLabel(row.original.source, row.original.extension_for)}
        </span>
      ),
    },
    {
      id: "last_opened_at",
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
      meta: { cellClassName: "max-w-[28rem]" },
    },
    {
      id: "hash",
      header: () => "Hash",
      enableSorting: false,
      cell: ({ row }) => {
        const paths = installedPathsFor(row.original.installed_versions ?? []);
        return <span className="text-muted-foreground font-mono text-xs">{singleHash(paths)}</span>;
      },
    },
  ];

  if (query.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to load software</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
        <Button variant="outline" size="sm" onClick={() => query.refetch()} className="mt-2 w-fit">
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
          <div className="relative max-w-md flex-1">
            <Search
              className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2"
              aria-hidden
            />
            <Input
              value={draft}
              onChange={(e) => setDraftDebounced(e.target.value)}
              placeholder="Search software"
              className="pr-8 pl-8"
              aria-label="Search host software"
            />
            {draft ? (
              <button
                type="button"
                onClick={() => {
                  setDraft("");
                  setActiveQuery("");
                  setPage(1);
                }}
                className="text-muted-foreground hover:text-foreground absolute top-1/2 right-2 -translate-y-1/2 rounded p-0.5"
                aria-label="Clear search"
              >
                <X className="size-3.5" />
              </button>
            ) : null}
          </div>
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

let debounceTimer: number = 0;

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
      <span className="text-muted-foreground font-mono text-xs break-all" title={paths[0].path}>
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
