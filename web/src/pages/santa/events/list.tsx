import { getRouteApi, Link } from "@tanstack/react-router";
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import { Activity } from "lucide-react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { FilterChip } from "@/components/filter-controls";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { PathText } from "@/components/path-text";
import { QueryError } from "@/components/query-error";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDataTable } from "@/hooks/use-data-table";
import { useDataTableSearch } from "@/hooks/use-data-table-search";
import { useHost } from "@/hooks/use-hosts";
import { useSantaEvents, useSantaFileAccessEvents } from "@/hooks/use-santa-events";
import type {
  SantaExecutionEvent as SantaEvent,
  SantaFileAccessEvent,
  SantaHostSummary,
} from "@/lib/api";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { formatDateTime, formatRelative } from "@/lib/utils";

import { DECISION_FILTERS, FILE_ACCESS_DECISION_FILTERS, fileName } from "./decisions";
import { ExecutionDecisionBadge, FileAccessDecisionBadge } from "./event-ui";
const executionRouteApi = getRouteApi("/_authenticated/santa/events/");
const fileAccessRouteApi = getRouteApi("/_authenticated/santa/events/file-access/");

interface ExecutionEventTableRow {
  event: SantaEvent;
  hostFilterID: number | undefined;
}

function ExecutionUserCell({ row }: CellContext<ExecutionEventTableRow, unknown>) {
  return (
    <EventUserLink user={row.original.event.executing_user} hostId={row.original.hostFilterID} />
  );
}

const executionEventColumns: ColumnDef<ExecutionEventTableRow>[] = [
  {
    id: "occurred_at",
    accessorFn: (row) => row.event.occurred_at,
    header: "Occurred",
    cell: ({ row }) => <Timestamp value={row.original.event.occurred_at} />,
    meta: { label: "Occurred" },
  },
  {
    id: "file_name",
    accessorFn: (row) => row.event.executable.file_name,
    header: "Executable",
    cell: ({ row }) => (
      <Link
        to="/santa/events/$id"
        params={{ id: String(row.original.event.id) }}
        className="font-medium hover:underline"
      >
        {row.original.event.executable.file_name || "-"}
      </Link>
    ),
    enableHiding: false,
    meta: { label: "Executable" },
  },
  {
    id: "file_path",
    accessorFn: (row) => row.event.file_path,
    enableSorting: false,
    header: "Path",
    cell: ({ row }) => <PathText value={row.original.event.file_path} />,
    meta: { label: "Path" },
  },
  {
    id: "decision",
    accessorFn: (row) => row.event.decision,
    header: "Decision",
    cell: ({ row }) => <ExecutionDecisionBadge decision={row.original.event.decision} />,
    meta: { label: "Decision", options: DECISION_FILTERS },
    enableColumnFilter: true,
  },
  {
    id: "host",
    accessorFn: (row) => row.event.host.display_name,
    header: "Host",
    cell: ({ row }) => <EventHostLink host={row.original.event.host} />,
    meta: { label: "Host" },
  },
  {
    id: "executing_user",
    accessorFn: (row) => row.event.executing_user,
    header: "User",
    cell: ExecutionUserCell,
    meta: { label: "User" },
  },
];

const fileAccessEventColumns: ColumnDef<SantaFileAccessEvent>[] = [
  {
    id: "occurred_at",
    accessorKey: "occurred_at",
    header: "Occurred",
    cell: ({ row }) => <Timestamp value={row.original.occurred_at} />,
    meta: { label: "Occurred" },
  },
  {
    id: "target",
    accessorKey: "target",
    header: "Target",
    cell: ({ row }) => (
      <Link
        to="/santa/events/file-access/$id"
        params={{ id: String(row.original.id) }}
        className="font-medium hover:underline"
      >
        {fileName(row.original.target) || row.original.target}
      </Link>
    ),
    enableHiding: false,
    meta: { label: "Target" },
  },
  {
    id: "decision",
    accessorKey: "decision",
    header: "Decision",
    cell: ({ row }) => <FileAccessDecisionBadge decision={row.original.decision} />,
    meta: { label: "Decision", options: FILE_ACCESS_DECISION_FILTERS },
    enableColumnFilter: true,
  },
  {
    id: "host",
    accessorFn: (row) => row.host.display_name,
    header: "Host",
    cell: ({ row }) => <EventHostLink host={row.original.host} />,
    meta: { label: "Host" },
  },
  {
    id: "process",
    enableSorting: false,
    header: "Process",
    cell: ({ row }) => row.original.primary_process.file_name || "-",
    meta: { label: "Process" },
  },
  {
    id: "rule_name",
    accessorKey: "rule_name",
    header: "Rule",
    cell: ({ row }) => row.original.rule_name || "-",
    meta: { label: "Rule" },
  },
  {
    id: "rule_version",
    accessorKey: "rule_version",
    header: "Rule Version",
    cell: ({ row }) => row.original.rule_version || "-",
    enableSorting: false,
    meta: { label: "Rule Version" },
  },
];

export function SantaEventListPage() {
  const search = executionRouteApi.useSearch();
  const navigate = executionRouteApi.useNavigate();
  return (
    <PageShell>
      <PageHeader
        title="Events"
        description="Review Santa execution and file access activity."
        context={
          <EventContextChips
            hostId={search.host_id ?? null}
            user={search.user}
            onClearHost={() =>
              void navigate({ search: (previous) => ({ ...previous, host_id: undefined }) })
            }
            onClearUser={() =>
              void navigate({ search: (previous) => ({ ...previous, user: undefined }) })
            }
          />
        }
      />
      <EventListNav active="execution" hostId={search.host_id ?? null} />
      <ExecutionEventsTable hostId={search.host_id} user={search.user} />
    </PageShell>
  );
}
export function SantaFileAccessEventListPage() {
  const search = fileAccessRouteApi.useSearch();
  const navigate = fileAccessRouteApi.useNavigate();
  return (
    <PageShell>
      <PageHeader
        title="Events"
        description="Review Santa execution and file access activity."
        context={
          <EventContextChips
            hostId={search.host_id ?? null}
            onClearHost={() =>
              void navigate({ search: (previous) => ({ ...previous, host_id: undefined }) })
            }
          />
        }
      />
      <EventListNav active="file-access" hostId={search.host_id ?? null} />
      <FileAccessEventsTable hostId={search.host_id} />
    </PageShell>
  );
}
function EventContextChips({
  hostId,
  user,
  onClearHost,
  onClearUser,
}: {
  hostId: number | null;
  user?: string | null;
  onClearHost: () => void;
  onClearUser?: () => void;
}) {
  const host = useHost(hostId);
  return (
    <>
      {hostId != null ? (
        <FilterChip
          label="Host"
          value={host.data?.display_name ?? `#${hostId}`}
          onRemove={onClearHost}
        />
      ) : null}
      {user && onClearUser ? <FilterChip label="User" value={user} onRemove={onClearUser} /> : null}
    </>
  );
}
function EventListNav({
  active,
  hostId,
}: {
  active: "execution" | "file-access";
  hostId: number | null;
}) {
  const search = hostId != null ? { host_id: hostId } : {};
  return (
    <Tabs value={active}>
      <TabsList>
        <TabsTrigger
          value="execution"
          render={<Link to="/santa/events" search={search} />}
          nativeButton={false}
        >
          Execution
        </TabsTrigger>
        <TabsTrigger
          value="file-access"
          render={<Link to="/santa/events/file-access" search={search} />}
          nativeButton={false}
        >
          File Access
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}
function ExecutionEventsTable({ hostId, user }: { hostId?: number; user?: string }) {
  const search = executionRouteApi.useSearch();
  const navigate = executionRouteApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: [{ id: "decision", multiple: true }],
    scopeKeys: ["host_id", "user"],
  });
  const decisions = search.decision ?? [];
  const query = useSantaEvents({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    host_id: hostId,
    user,
    decisions,
  });
  const events = query.data?.items ?? [];
  const tableRows: ExecutionEventTableRow[] = events.map((event) => ({
    event,
    hostFilterID: hostId,
  }));
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: tableRows,
    columns: executionEventColumns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.event.id),
  });
  if (query.error) {
    return (
      <QueryError
        title="Failed to load execution events"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  if (query.isLoading) {
    return <DataTableSkeleton columnCount={6} filterCount={1} />;
  }
  return (
    <DataTable
      table={table}
      empty={<EventsEmptyState hasFilters={tableSearch.isFiltered} noun="execution events" />}
    >
      <div className="flex flex-wrap items-center gap-2 p-1">
        <DataTableSearchInput
          className="h-8 w-56 lg:w-72"
          placeholder="Search executable, path, host, user"
          value={tableSearch.q ?? ""}
          onValueChange={tableSearch.onQueryChange}
        />
        <DataTableFacetedFilter
          column={table.getColumn("decision")}
          title="Decision"
          options={DECISION_FILTERS}
          multiple
        />
      </div>
    </DataTable>
  );
}
function FileAccessEventsTable({ hostId }: { hostId?: number }) {
  const search = fileAccessRouteApi.useSearch();
  const navigate = fileAccessRouteApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: [{ id: "decision", multiple: true }],
    scopeKeys: ["host_id"],
  });
  const decisions = search.decision ?? [];
  const query = useSantaFileAccessEvents({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    host_id: hostId,
    decisions,
  });
  const events = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const table = useDataTable({
    tableState: tableSearch,
    data: events,
    columns: fileAccessEventColumns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
  });
  if (query.error) {
    return (
      <QueryError
        title="Failed to load file access events"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  if (query.isLoading) {
    return <DataTableSkeleton columnCount={6} filterCount={1} />;
  }
  return (
    <DataTable
      table={table}
      empty={<EventsEmptyState hasFilters={tableSearch.isFiltered} noun="file access events" />}
    >
      <div className="flex flex-wrap items-center gap-2 p-1">
        <DataTableSearchInput
          className="h-8 w-56 lg:w-72"
          placeholder="Search target, process, host, signer"
          value={tableSearch.q ?? ""}
          onValueChange={tableSearch.onQueryChange}
        />
        <DataTableFacetedFilter
          column={table.getColumn("decision")}
          title="Decision"
          options={FILE_ACCESS_DECISION_FILTERS}
          multiple
        />
      </div>
    </DataTable>
  );
}
function EventsEmptyState({ hasFilters, noun }: { hasFilters: boolean; noun: string }) {
  return (
    <DataTableEmpty
      icon={<Activity />}
      filtered={hasFilters}
      title={`No ${noun}`}
      description="Client decisions appear after Santa syncs."
      filteredDescription="No events matched these filters."
    />
  );
}
function EventHostLink({ host }: { host: SantaHostSummary }) {
  return (
    <Link to="/hosts/$id" params={{ id: String(host.id) }} className="hover:underline">
      {host.display_name}
    </Link>
  );
}
function EventUserLink({ user, hostId }: { user: string; hostId?: number }) {
  if (!user) return "-";
  return (
    <Link to="/santa/events" search={{ host_id: hostId, user }} className="hover:underline">
      {user}
    </Link>
  );
}
function Timestamp({ value }: { value: string }) {
  return <span title={formatDateTime(value)}>{formatRelative(value)}</span>;
}
