import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Activity } from "lucide-react";
import { parseAsInteger, parseAsString, useQueryStates } from "nuqs";
import * as React from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { FilterChip } from "@/components/filter-controls";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDataTable } from "@/hooks/use-data-table";
import { useDataTableSearch } from "@/hooks/use-data-table-search";
import { useHost } from "@/hooks/use-hosts";
import {
  type SantaEventListParams,
  type SantaFileAccessEventListParams,
  useSantaEvents,
  useSantaFileAccessEvents,
} from "@/hooks/use-santa-events";
import type {
  SantaExecutionEvent as SantaEvent,
  SantaFileAccessEvent,
  SantaHostSummary,
} from "@/lib/api";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { formatDateTime, formatRelative } from "@/lib/utils";

import { DECISION_FILTERS, FILE_ACCESS_DECISION_FILTERS, fileName } from "./decisions";
import { ExecutionDecisionBadge, FileAccessDecisionBadge } from "./event-ui";

const DECISION_FILTER_KEYS = [{ id: "decision" }] as const;

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
        <TabsTrigger value="execution" asChild>
          <Link to="/santa/events" search={search}>
            Execution
          </Link>
        </TabsTrigger>
        <TabsTrigger value="file-access" asChild>
          <Link to="/santa/events/file-access" search={search}>
            File Access
          </Link>
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}

function ExecutionEventsTable({ hostId, user }: { hostId?: number; user?: string }) {
  const tableSearch = useDataTableSearch(DECISION_FILTER_KEYS);
  const decisions = tableSearch.filters.decision ?? [];

  const query = useSantaEvents({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    host_id: hostId,
    user,
    decisions: decisions as SantaEventListParams["decisions"],
  });

  const events = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters =
    !!tableSearch.q || hostId !== undefined || user !== undefined || decisions.length > 0;

  const columns = React.useMemo<ColumnDef<SantaEvent>[]>(
    () => [
      {
        id: "occurred_at",
        accessorKey: "occurred_at",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Occurred" />,
        cell: ({ row }) => <Timestamp value={row.original.occurred_at} />,
        meta: { label: "Occurred" },
      },
      {
        id: "file_name",
        accessorFn: (row) => row.executable.file_name || row.executable.sha256,
        header: ({ column }) => <DataTableColumnHeader column={column} label="Executable" />,
        cell: ({ row }) => (
          <Link
            to="/santa/events/$eventId"
            params={{ eventId: String(row.original.id) }}
            className="font-medium hover:underline"
          >
            {row.original.executable.file_name || row.original.executable.sha256}
          </Link>
        ),
        enableHiding: false,
        meta: { label: "Executable" },
      },
      {
        id: "file_path",
        accessorKey: "file_path",
        enableSorting: false,
        header: ({ column }) => <DataTableColumnHeader column={column} label="Path" />,
        cell: ({ row }) => row.original.file_path || "-",
        meta: { label: "Path" },
      },
      {
        id: "decision",
        accessorKey: "decision",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Decision" />,
        cell: ({ row }) => <ExecutionDecisionBadge decision={row.original.decision} />,
        meta: { label: "Decision", options: DECISION_FILTERS },
        enableColumnFilter: true,
      },
      {
        id: "host",
        accessorFn: (row) => row.host.display_name,
        header: ({ column }) => <DataTableColumnHeader column={column} label="Host" />,
        cell: ({ row }) => <EventHostLink host={row.original.host} />,
        meta: { label: "Host" },
      },
      {
        id: "executing_user",
        accessorKey: "executing_user",
        header: ({ column }) => <DataTableColumnHeader column={column} label="User" />,
        cell: ({ row }) => <EventUserLink user={row.original.executing_user} hostId={hostId} />,
        meta: { label: "User" },
      },
    ],
    [hostId],
  );

  const table = useDataTable({
    tableState: tableSearch,
    data: events,
    columns,
    pageCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.id),
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
  const decisions = tableSearch.filters.decision ?? [];

  const query = useSantaFileAccessEvents({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    host_id: hostId,
    decisions: decisions as SantaFileAccessEventListParams["decisions"],
  });

  const events = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q || hostId !== undefined || decisions.length > 0;

  const columns = React.useMemo<ColumnDef<SantaFileAccessEvent>[]>(
    () => [
      {
        id: "occurred_at",
        accessorKey: "occurred_at",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Occurred" />,
        cell: ({ row }) => <Timestamp value={row.original.occurred_at} />,
        meta: { label: "Occurred" },
      },
      {
        id: "target",
        accessorKey: "target",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Target" />,
        cell: ({ row }) => (
          <Link
            to="/santa/events/file-access/$eventId"
            params={{ eventId: String(row.original.id) }}
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
        header: ({ column }) => <DataTableColumnHeader column={column} label="Decision" />,
        cell: ({ row }) => <FileAccessDecisionBadge decision={row.original.decision} />,
        meta: { label: "Decision", options: FILE_ACCESS_DECISION_FILTERS },
        enableColumnFilter: true,
      },
      {
        id: "host",
        accessorFn: (row) => row.host.display_name,
        header: ({ column }) => <DataTableColumnHeader column={column} label="Host" />,
        cell: ({ row }) => <EventHostLink host={row.original.host} />,
        meta: { label: "Host" },
      },
      {
        id: "process",
        enableSorting: false,
        header: ({ column }) => <DataTableColumnHeader column={column} label="Process" />,
        cell: ({ row }) =>
          row.original.primary_process.file_name ||
          fileName(row.original.primary_process.file_path) ||
          "-",
        meta: { label: "Process" },
      },
      {
        id: "rule_name",
        accessorKey: "rule_name",
        header: ({ column }) => <DataTableColumnHeader column={column} label="Rule" />,
        cell: ({ row }) => row.original.rule_name || row.original.rule_version || "-",
        meta: { label: "Rule" },
      },
    ],
    [],
  );

  const table = useDataTable({
    tableState: tableSearch,
    data: events,
    columns,
    pageCount,
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
    <Link to="/hosts/$hostId" params={{ hostId: String(host.id) }} className="hover:underline">
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
