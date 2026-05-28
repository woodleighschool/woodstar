import { useNavigate, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Activity } from "lucide-react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmptyState } from "@/components/data-table/data-table-empty-state";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import {
  useSantaEvents,
  useSantaFileAccessEvents,
  type SantaEvent,
  type SantaFileAccessEvent,
} from "@/hooks/use-santa";
import { tableQueryParams, useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { DECISION_FILTERS, FILE_ACCESS_DECISION_FILTERS, fileName } from "./constants";
import { DecisionBadge, HostLink, Timestamp } from "./event-ui";

const EVENT_TYPE_EXECUTION = "execution";
const EVENT_TYPE_FILE_ACCESS = "file_access";

type EventType = typeof EVENT_TYPE_EXECUTION | typeof EVENT_TYPE_FILE_ACCESS;

export function SantaEventsPage() {
  const search = useSearch({ strict: false });
  const navigate = useNavigate();
  const eventType = search.event_type === EVENT_TYPE_FILE_ACCESS ? EVENT_TYPE_FILE_ACCESS : EVENT_TYPE_EXECUTION;

  const setEventType = (next: string) => {
    void navigate({
      search: ((prev: Record<string, unknown>) => ({
        q: prev.q,
        event_type: next === EVENT_TYPE_FILE_ACCESS ? EVENT_TYPE_FILE_ACCESS : undefined,
      })) as never,
      replace: true,
    });
  };

  return (
    <PageShell>
      <PageHeader title="Events" />

      <Tabs value={eventType} onValueChange={setEventType}>
        <TabsList>
          <TabsTrigger value={EVENT_TYPE_EXECUTION}>Execution</TabsTrigger>
          <TabsTrigger value={EVENT_TYPE_FILE_ACCESS}>File access</TabsTrigger>
        </TabsList>

        <TabsContent value={EVENT_TYPE_EXECUTION}>
          {eventType === EVENT_TYPE_EXECUTION ? <ExecutionEventsTable eventType={eventType} /> : null}
        </TabsContent>
        <TabsContent value={EVENT_TYPE_FILE_ACCESS}>
          {eventType === EVENT_TYPE_FILE_ACCESS ? <FileAccessEventsTable eventType={eventType} /> : null}
        </TabsContent>
      </Tabs>
    </PageShell>
  );
}

function ExecutionEventsTable({ eventType }: { eventType: EventType }) {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q", { resetKeys: ["page_index"] });
  const decisions = search.decisions ?? [];
  const query = useSantaEvents({
    q: search.q,
    decisions,
    ...tableQueryParams(state),
  });
  const rows = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || decisions.length > 0;

  const columns: ColumnDef<SantaEvent>[] = [
    {
      id: "file_name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Executable" />,
      cell: ({ row }) => (
        <div className="grid gap-1">
          <span className="font-medium">{row.original.executable.file_name || row.original.executable.sha256}</span>
          <span className="max-w-[34rem] truncate text-xs">
            {row.original.file_path || row.original.executable.sha256}
          </span>
        </div>
      ),
    },
    {
      id: "decision",
      accessorKey: "decision",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Decision" />,
      cell: ({ row }) => <DecisionBadge decision={row.original.decision} />,
    },
    {
      id: "host_id",
      accessorFn: (row) => row.host.display_name,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
      cell: ({ row }) => <HostLink host={row.original.host} />,
    },
    {
      id: "executing_user",
      accessorKey: "executing_user",
      header: ({ column }) => <DataTableColumnHeader column={column} title="User" />,
      cell: ({ row }) => row.original.executing_user || "-",
    },
    {
      id: "occurred_at",
      accessorFn: (row) => row.occurred_at,
      header: ({ column }) => <DataTableColumnHeader column={column} title="When" />,
      cell: ({ row }) => <Timestamp value={row.original.occurred_at} />,
    },
  ];

  return (
    <>
      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load execution events</AlertTitle>
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
          rowHref={(row) => ({ to: "/santa/events/$eventId", params: { eventId: String(row.id) } })}
          toolbar={
            <EventTableToolbar
              draft={draft}
              setDraft={setDraft}
              decisions={decisions}
              decisionOptions={[...DECISION_FILTERS]}
              onDecisionsChange={(next) => setters.setFilter("decisions", next.length > 0 ? next.join(",") : undefined)}
              eventType={eventType}
            />
          }
          empty={
            <DataTableEmptyState
              icon={<Activity />}
              title={hasFilters ? "No matches" : "No execution events"}
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

function FileAccessEventsTable({ eventType }: { eventType: EventType }) {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q", { resetKeys: ["page_index"] });
  const decisions = search.decisions ?? [];
  const query = useSantaFileAccessEvents({
    q: search.q,
    decisions,
    ...tableQueryParams(state),
  });
  const rows = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || decisions.length > 0;

  const columns: ColumnDef<SantaFileAccessEvent>[] = [
    {
      id: "target",
      accessorKey: "target",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Target" />,
      cell: ({ row }) => (
        <span className="block max-w-96 truncate font-medium" title={row.original.target}>
          {fileName(row.original.target) || row.original.target}
        </span>
      ),
    },
    {
      id: "decision",
      accessorKey: "decision",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Decision" />,
      cell: ({ row }) => <DecisionBadge decision={row.original.decision} />,
    },
    {
      id: "host",
      accessorFn: (row) => row.host.display_name,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
      cell: ({ row }) => <HostLink host={row.original.host} />,
    },
    {
      id: "process",
      header: "Process",
      enableSorting: false,
      cell: ({ row }) => (
        <span className="block max-w-60 truncate">
          {row.original.primary_process.file_name || fileName(row.original.primary_process.file_path) || "-"}
        </span>
      ),
    },
    {
      id: "rule_name",
      accessorKey: "rule_name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Rule" />,
      cell: ({ row }) => <span>{row.original.rule_name || row.original.rule_version || "-"}</span>,
    },
    {
      id: "occurred_at",
      accessorFn: (row) => row.occurred_at,
      header: ({ column }) => <DataTableColumnHeader column={column} title="Occurred" />,
      cell: ({ row }) => <Timestamp value={row.original.occurred_at} />,
    },
  ];

  return (
    <>
      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load file access events</AlertTitle>
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
          rowHref={(row) => ({ to: "/santa/file-access-events/$eventId", params: { eventId: String(row.id) } })}
          toolbar={
            <EventTableToolbar
              draft={draft}
              setDraft={setDraft}
              decisions={decisions}
              decisionOptions={[...FILE_ACCESS_DECISION_FILTERS]}
              onDecisionsChange={(next) => setters.setFilter("decisions", next.length > 0 ? next.join(",") : undefined)}
              eventType={eventType}
            />
          }
          empty={
            <DataTableEmptyState
              icon={<Activity />}
              title={hasFilters ? "No matches" : "No file access events"}
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

function EventTableToolbar({
  draft,
  setDraft,
  decisions,
  decisionOptions,
  onDecisionsChange,
  eventType,
}: {
  draft: string;
  setDraft: (next: string) => void;
  decisions: string[];
  decisionOptions: Array<{ value: string; label: string }>;
  onDecisionsChange: (next: string[]) => void;
  eventType: EventType;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <DataTableSearch
        value={draft}
        onChange={setDraft}
        placeholder={eventType === EVENT_TYPE_FILE_ACCESS ? "Search target, process, host, signer" : "Search"}
        label={eventType === EVENT_TYPE_FILE_ACCESS ? "Search file access events" : "Search execution events"}
      />
      <DataTableFacetedFilter
        title="Decision"
        options={decisionOptions}
        selected={decisions}
        onChange={onDecisionsChange}
      />
    </div>
  );
}
