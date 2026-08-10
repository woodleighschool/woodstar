import { getRouteApi } from "@tanstack/react-router";
import { Activity } from "lucide-react";
import { useMemo } from "react";

import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import type { DataTableCellContext, DataTableColumnDef } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { FilterChip } from "@components/filter-controls";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@components/layout/scrollable-tabs";
import { Link, TextLink } from "@components/link";
import { QueryError } from "@components/query-error";
import { TabsTrigger } from "@components/ui/tabs";
import { useHost } from "@features/hosts/queries";
import type { SantaExecutionEvent, SantaFileAccessEvent, SantaHostSummary } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";
import { formatDateTime, formatRelative } from "@lib/utils";

import { DECISION_FILTERS, FILE_ACCESS_DECISION_FILTERS, fileName } from "./decisions";
import { ExecutionDecisionBadge, FileAccessDecisionBadge } from "./event-ui";
import { useSantaEvents, useSantaFileAccessEvents } from "./queries";
const executionRouteApi = getRouteApi("/_authenticated/santa/events/");
const fileAccessRouteApi = getRouteApi("/_authenticated/santa/events/file-access/");

interface ExecutionEventTableRow {
  event: SantaExecutionEvent;
  hostFilterID: number | undefined;
}

function ExecutionUserCell({ row }: DataTableCellContext<ExecutionEventTableRow>) {
  return (
    <EventUserLink user={row.original.event.executing_user} hostId={row.original.hostFilterID} />
  );
}

const executionEventColumns: DataTableColumnDef<ExecutionEventTableRow>[] = [
  {
    id: "occurred_at",
    accessorFn: (row) => row.event.occurred_at,
    header: "Occurred",
    cell: ({ row }) => <Timestamp value={row.original.event.occurred_at} />,
    size: 144,
    meta: { label: "Occurred" },
  },
  {
    id: "file_name",
    accessorFn: (row) => row.event.executable.file_name,
    header: "Executable",
    cell: ({ row }) => (
      <TextLink
        to="/santa/events/$id"
        params={{ id: String(row.original.event.id) }}
        className="font-medium"
      >
        {row.original.event.executable.file_name || "-"}
      </TextLink>
    ),
    enableHiding: false,
    size: 200,
    minSize: 120,
    meta: { label: "Executable" },
  },
  {
    id: "file_path",
    accessorFn: (row) => row.event.file_path,
    enableSorting: false,
    header: "Path",
    cell: ({ row }) => row.original.event.file_path || "-",
    size: 360,
    minSize: 200,
    meta: { label: "Path" },
  },
  {
    id: "decision",
    accessorFn: (row) => row.event.decision,
    header: "Decision",
    cell: ({ row }) => <ExecutionDecisionBadge decision={row.original.event.decision} />,
    meta: { label: "Decision", options: DECISION_FILTERS },
    enableColumnFilter: true,
    size: 152,
    minSize: 136,
  },
  {
    id: "host",
    accessorFn: (row) => row.event.host.display_name,
    header: "Host",
    cell: ({ row }) => <EventHostLink host={row.original.event.host} />,
    size: 160,
    meta: { label: "Host" },
  },
  {
    id: "executing_user",
    accessorFn: (row) => row.event.executing_user,
    header: "User",
    cell: ExecutionUserCell,
    size: 144,
    meta: { label: "User" },
  },
];

const fileAccessEventColumns: DataTableColumnDef<SantaFileAccessEvent>[] = [
  {
    id: "occurred_at",
    accessorKey: "occurred_at",
    header: "Occurred",
    cell: ({ row }) => <Timestamp value={row.original.occurred_at} />,
    size: 144,
    meta: { label: "Occurred" },
  },
  {
    id: "target",
    accessorKey: "target",
    header: "Target",
    cell: ({ row }) => (
      <TextLink
        to="/santa/events/file-access/$id"
        params={{ id: String(row.original.id) }}
        className="font-medium"
      >
        {fileName(row.original.target) || row.original.target}
      </TextLink>
    ),
    enableHiding: false,
    size: 280,
    minSize: 160,
    meta: { label: "Target" },
  },
  {
    id: "decision",
    accessorKey: "decision",
    header: "Decision",
    cell: ({ row }) => <FileAccessDecisionBadge decision={row.original.decision} />,
    meta: { label: "Decision", options: FILE_ACCESS_DECISION_FILTERS },
    enableColumnFilter: true,
    size: 152,
    minSize: 136,
  },
  {
    id: "host",
    accessorFn: (row) => row.host.display_name,
    header: "Host",
    cell: ({ row }) => <EventHostLink host={row.original.host} />,
    size: 160,
    meta: { label: "Host" },
  },
  {
    id: "process",
    enableSorting: false,
    header: "Process",
    cell: ({ row }) => row.original.primary_process.file_name || "-",
    size: 200,
    meta: { label: "Process" },
  },
  {
    id: "rule_name",
    accessorKey: "rule_name",
    header: "Rule",
    cell: ({ row }) => row.original.rule_name || "-",
    size: 180,
    meta: { label: "Rule" },
  },
  {
    id: "rule_version",
    accessorKey: "rule_version",
    header: "Rule Version",
    cell: ({ row }) => row.original.rule_version || "-",
    enableSorting: false,
    size: 112,
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
              void navigate({
                search: (previous) => ({ ...previous, host_id: undefined }),
              })
            }
            onClearUser={() =>
              void navigate({
                search: (previous) => ({ ...previous, user: undefined }),
              })
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
              void navigate({
                search: (previous) => ({ ...previous, host_id: undefined }),
              })
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
    <ScrollableTabs value={active}>
      <ScrollableTabsList>
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
      </ScrollableTabsList>
    </ScrollableTabs>
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
  const tableRows = useMemo<ExecutionEventTableRow[]>(
    () =>
      query.data?.items.map((event) => ({
        event,
        hostFilterID: hostId,
      })) ?? [],
    [hostId, query.data?.items],
  );
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
      pending={query.isPlaceholderData}
      empty={<EventsEmptyState hasFilters={tableSearch.isFiltered} noun="execution events" />}
    >
      <DataTableSearchInput
        loading={query.isPlaceholderData}
        placeholder="Search executable, path, host, user"
        value={tableSearch.q ?? ""}
        onValueChange={tableSearch.onQueryChange}
      />
      <DataTableFacetedFilter
        column={table.getColumn("decision")}
        title="Decision"
        options={DECISION_FILTERS}
      />
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
  const events = useMemo(() => query.data?.items ?? [], [query.data?.items]);
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
      pending={query.isPlaceholderData}
      empty={<EventsEmptyState hasFilters={tableSearch.isFiltered} noun="file access events" />}
    >
      <DataTableSearchInput
        loading={query.isPlaceholderData}
        placeholder="Search target, process, host, signer"
        value={tableSearch.q ?? ""}
        onValueChange={tableSearch.onQueryChange}
      />
      <DataTableFacetedFilter
        column={table.getColumn("decision")}
        title="Decision"
        options={FILE_ACCESS_DECISION_FILTERS}
      />
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
    <TextLink to="/hosts/$id" params={{ id: String(host.id) }}>
      {host.display_name}
    </TextLink>
  );
}
function EventUserLink({ user, hostId }: { user: string; hostId?: number }) {
  if (!user) return "-";
  return (
    <TextLink to="/santa/events" search={{ host_id: hostId, user }}>
      {user}
    </TextLink>
  );
}
function Timestamp({ value }: { value: string }) {
  return <span title={formatDateTime(value)}>{formatRelative(value)}</span>;
}
