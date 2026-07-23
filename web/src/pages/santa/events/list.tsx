import { Link } from "@tanstack/react-router";
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import { Activity } from "lucide-react";
import { parseAsInteger, parseAsString, useQueryStates } from "nuqs";

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
import { formatDateTime, formatRelative, isOneOf } from "@/lib/utils";

import {
  DECISION_FILTERS,
  DECISION_FILTER_VALUES,
  FILE_ACCESS_DECISION_FILTERS,
  FILE_ACCESS_DECISION_VALUES,
  fileName,
} from "./decisions";
import { ExecutionDecisionBadge, FileAccessDecisionBadge } from "./event-ui";
const DECISION_FILTER_KEYS = [{ id: "decision" }] as const;

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
  const [deepLink, setDeepLink] = useQueryStates({ host_id: parseAsInteger, user: parseAsString });
  return (
    <PageShell>
      <PageHeader
        title="Events"
        description="Review Santa execution and file access activity."
        context={
          <EventContextChips
            hostId={deepLink.host_id}
            user={deepLink.user}
            onClearHost={() => void setDeepLink({ host_id: null })}
            onClearUser={() => void setDeepLink({ user: null })}
          />
        }
      />
      <EventListNav active="execution" hostId={deepLink.host_id} />
      <ExecutionEventsTable
        hostId={deepLink.host_id ?? undefined}
        user={deepLink.user ?? undefined}
      />
    </PageShell>
  );
}
export function SantaFileAccessEventListPage() {
  const [deepLink, setDeepLink] = useQueryStates({ host_id: parseAsInteger });
  return (
    <PageShell>
      <PageHeader
        title="Events"
        description="Review Santa execution and file access activity."
        context={
          <EventContextChips
            hostId={deepLink.host_id}
            onClearHost={() => void setDeepLink({ host_id: null })}
          />
        }
      />
      <EventListNav active="file-access" hostId={deepLink.host_id} />
      <FileAccessEventsTable hostId={deepLink.host_id ?? undefined} />
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
  const tableSearch = useDataTableSearch(DECISION_FILTER_KEYS);
  const decisions = (tableSearch.filters.decision ?? []).filter((value) =>
    isOneOf(value, DECISION_FILTER_VALUES),
  );
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
  const hasFilters =
    !!tableSearch.q || hostId !== undefined || user !== undefined || decisions.length > 0;
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
      empty={<EventsEmptyState hasFilters={hasFilters} noun="execution events" />}
    >
      <div className="flex flex-wrap items-center gap-2 p-1">
        <DataTableSearchInput
          className="h-8 w-56 lg:w-72"
          placeholder="Search executable, path, host, user"
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
  const tableSearch = useDataTableSearch(DECISION_FILTER_KEYS);
  const decisions = (tableSearch.filters.decision ?? []).filter((value) =>
    isOneOf(value, FILE_ACCESS_DECISION_VALUES),
  );
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
  const hasFilters = !!tableSearch.q || hostId !== undefined || decisions.length > 0;
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
      empty={<EventsEmptyState hasFilters={hasFilters} noun="file access events" />}
    >
      <div className="flex flex-wrap items-center gap-2 p-1">
        <DataTableSearchInput
          className="h-8 w-56 lg:w-72"
          placeholder="Search target, process, host, signer"
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
