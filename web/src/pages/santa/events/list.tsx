import { Link, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Activity } from "lucide-react";
import type { ReactNode } from "react";

import {
  DataTable,
  DataTableColumnHeader,
  DataTableEmptyState,
  DataTableFacetedFilter,
  DataTableSearch,
} from "@/components/data-table";
import { FilterChip } from "@/components/filter-controls";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useHost, type HostDetail } from "@/hooks/use-hosts";
import {
  useSantaEvents,
  useSantaFileAccessEvents,
  type SantaEvent,
  type SantaFileAccessEvent,
  type SantaHostSummary,
} from "@/hooks/use-santa";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { formatRelative } from "@/lib/utils";
import { DECISION_FILTERS, FILE_ACCESS_DECISION_FILTERS, fileName } from "./constants";
import { ExecutionDecisionBadge, FileAccessDecisionBadge } from "./event-ui";

type EventListKind = "execution" | "file-access";

export function SantaEventsPage() {
  return (
    <PageShell>
      <PageHeader title="Events" description="Review Santa execution and file access activity." />
      <EventListNav active="execution" />
      <ExecutionEventsTable />
    </PageShell>
  );
}

export function SantaFileAccessEventsPage() {
  return (
    <PageShell>
      <PageHeader title="Events" description="Review Santa execution and file access activity." />
      <EventListNav active="file-access" />
      <FileAccessEventsTable />
    </PageShell>
  );
}

function EventListNav({ active }: { active: EventListKind }) {
  const search = useSearch({ strict: false });
  const sharedSearch = {
    q: typeof search.q === "string" ? search.q : undefined,
    host_id: typeof search.host_id === "number" ? search.host_id : undefined,
  };

  return (
    <Tabs value={active}>
      <TabsList>
        <TabsTrigger value="execution" asChild>
          <Link to="/santa/events" search={sharedSearch}>
            Execution
          </Link>
        </TabsTrigger>
        <TabsTrigger value="file-access" asChild>
          <Link to="/santa/events/file-access" search={sharedSearch}>
            File Access
          </Link>
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}

function ExecutionEventsTable() {
  const search = useSearch({ from: "/_authenticated/santa/events/" });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q", { resetKeys: ["page_index"] });
  const decisions = search.decisions ?? [];
  const hostID = typeof search.host_id === "number" ? search.host_id : undefined;
  const user = typeof search.user === "string" ? search.user : undefined;
  const host = useHost(hostID ?? null);
  const query = useSantaEvents({
    q: search.q,
    host_id: hostID,
    user,
    decisions,
    ...tableQueryParams(state),
  });
  const rows = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || !!hostID || !!user || decisions.length > 0;

  const columns: ColumnDef<SantaEvent>[] = [
    {
      id: "occurred_at",
      accessorKey: "occurred_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Occurred At" />,
      cell: ({ row }) => formatRelative(row.original.occurred_at),
      meta: {
        label: "Occurred At",
        cellClassName: "w-44",
      },
    },
    {
      id: "file_name",
      accessorFn: (row) => row.executable.file_name || row.executable.sha256,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Executable" />,
      cell: ({ row }) => row.original.executable.file_name || row.original.executable.sha256,
      meta: {
        label: "Executable",
      },
    },
    {
      id: "file_path",
      accessorKey: "file_path",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Path" />,
      enableSorting: false,
      cell: ({ row }) => row.original.file_path || "-",
      meta: {
        label: "Path",
      },
    },
    {
      id: "decision",
      accessorKey: "decision",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Decision" />,
      cell: ({ row }) => <ExecutionDecisionBadge decision={row.original.decision} />,
    },
    {
      id: "host",
      accessorFn: (row) => row.host.display_name,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
      cell: ({ row }) => <EventHostLink host={row.original.host} />,
    },
    {
      id: "executing_user",
      accessorKey: "executing_user",
      header: ({ column }) => <DataTableColumnHeader column={column} title="User" />,
      cell: ({ row }) => <EventUserLink user={row.original.executing_user} hostId={hostID} />,
    },
  ];

  return (
    <>
      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Execution Events</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          totalCount={totalCount}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          showExport
          exportFilename="santa-execution-events.csv"
          rowHref={(row) => ({ to: "/santa/events/$eventId", params: { eventId: String(row.id) } })}
          toolbar={(_table, exportButton) => (
            <EventTableToolbar
              draft={draft}
              setDraft={setDraft}
              decisions={decisions}
              decisionOptions={[...DECISION_FILTERS]}
              onDecisionsChange={(next) => setters.setFilter("decisions", next.length > 0 ? next.join(",") : undefined)}
              hostName={host.data ? hostLabel(host.data) : undefined}
              onClearHost={() => setters.setFilter("host_id", undefined)}
              user={user}
              onClearUser={() => setters.setFilter("user", undefined)}
              searchPlaceholder="Search Executable, Path, Host, User"
              actions={exportButton}
            />
          )}
          empty={
            <DataTableEmptyState
              icon={<Activity />}
              title={hasFilters ? "No Matches" : "No Execution Events"}
              description={
                hasFilters ? "No events matched these filters." : "Client decisions appear after Santa syncs."
              }
            />
          }
        />
      )}
    </>
  );
}

function FileAccessEventsTable() {
  const search = useSearch({ from: "/_authenticated/santa/events/file-access/" });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q", { resetKeys: ["page_index"] });
  const decisions = search.decisions ?? [];
  const hostID = typeof search.host_id === "number" ? search.host_id : undefined;
  const host = useHost(hostID ?? null);
  const query = useSantaFileAccessEvents({
    q: search.q,
    host_id: hostID,
    decisions,
    ...tableQueryParams(state),
  });
  const rows = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || !!hostID || decisions.length > 0;

  const columns: ColumnDef<SantaFileAccessEvent>[] = [
    {
      id: "occurred_at",
      accessorKey: "occurred_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Occurred At" />,
      cell: ({ row }) => formatRelative(row.original.occurred_at),
      meta: {
        label: "Occurred At",
        cellClassName: "w-44",
      },
    },
    {
      id: "target",
      accessorKey: "target",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Target" />,
      cell: ({ row }) => fileName(row.original.target) || row.original.target,
    },
    {
      id: "decision",
      accessorKey: "decision",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Decision" />,
      cell: ({ row }) => <FileAccessDecisionBadge decision={row.original.decision} />,
    },
    {
      id: "host",
      accessorFn: (row) => row.host.display_name,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
      cell: ({ row }) => <EventHostLink host={row.original.host} />,
    },
    {
      id: "process",
      header: "Process",
      enableSorting: false,
      cell: ({ row }) =>
        row.original.primary_process.file_name || fileName(row.original.primary_process.file_path) || "-",
    },
    {
      id: "rule_name",
      accessorKey: "rule_name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Rule" />,
      cell: ({ row }) => row.original.rule_name || row.original.rule_version || "-",
    },
  ];

  return (
    <>
      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to Load File Access Events</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          totalCount={totalCount}
          pagination={state.pagination}
          sorting={state.sorting}
          onPaginationChange={setters.setPagination}
          onSortingChange={setters.setSorting}
          isLoading={query.isLoading}
          showExport
          exportFilename="santa-file-access-events.csv"
          rowHref={(row) => ({ to: "/santa/events/file-access/$eventId", params: { eventId: String(row.id) } })}
          toolbar={(_table, exportButton) => (
            <EventTableToolbar
              draft={draft}
              setDraft={setDraft}
              decisions={decisions}
              decisionOptions={[...FILE_ACCESS_DECISION_FILTERS]}
              onDecisionsChange={(next) => setters.setFilter("decisions", next.length > 0 ? next.join(",") : undefined)}
              hostName={host.data ? hostLabel(host.data) : undefined}
              onClearHost={() => setters.setFilter("host_id", undefined)}
              searchPlaceholder="Search Target, Process, Host, Signer"
              actions={exportButton}
            />
          )}
          empty={
            <DataTableEmptyState
              icon={<Activity />}
              title={hasFilters ? "No Matches" : "No File Access Events"}
              description={
                hasFilters
                  ? "No file access events matched these filters."
                  : "File access decisions appear after Santa syncs."
              }
            />
          }
        />
      )}
    </>
  );
}

function EventHostLink({ host }: { host: SantaHostSummary }) {
  return (
    <Link to="/hosts/$hostId" params={{ hostId: String(host.id) }} className="hover:underline">
      {host.display_name}
    </Link>
  );
}

function EventUserLink({ user, hostId }: { user: string; hostId: number | undefined }) {
  if (!user) return "-";
  return (
    <Link
      to="/santa/events"
      search={{ host_id: hostId, user }}
      className="hover:underline"
      onClick={(event) => event.stopPropagation()}
    >
      {user}
    </Link>
  );
}

function hostLabel(host: HostDetail) {
  return host.display_name;
}

function EventTableToolbar({
  draft,
  setDraft,
  decisions,
  decisionOptions,
  onDecisionsChange,
  hostName,
  onClearHost,
  user,
  onClearUser,
  searchPlaceholder,
  actions,
}: {
  draft: string;
  setDraft: (next: string) => void;
  decisions: string[];
  decisionOptions: Array<{ value: string; label: string }>;
  onDecisionsChange: (next: string[]) => void;
  hostName?: string;
  onClearHost?: () => void;
  user?: string;
  onClearUser?: () => void;
  searchPlaceholder: string;
  actions?: ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <DataTableSearch value={draft} onChange={setDraft} placeholder={searchPlaceholder} />
      <DataTableFacetedFilter
        title="Decision"
        options={decisionOptions}
        selected={decisions}
        onChange={onDecisionsChange}
      />
      {hostName && onClearHost ? <FilterChip label="Host" value={hostName} onRemove={onClearHost} /> : null}
      {user && onClearUser ? <FilterChip label="User" value={user} onRemove={onClearUser} /> : null}
      {actions ? <div className="ml-auto">{actions}</div> : null}
    </div>
  );
}
