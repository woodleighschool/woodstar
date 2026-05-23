import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Activity } from "lucide-react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useSantaEvents, type SantaEvent } from "@/hooks/use-santa";
import { formatRelative } from "@/lib/utils";

const DECISION_FILTERS = [
  { value: "all", label: "All decisions" },
  { value: "blocked", label: "Blocked" },
  { value: "allowed", label: "Allowed" },
  { value: "allow_binary", label: "Allow binary" },
  { value: "allow_certificate", label: "Allow certificate" },
  { value: "block_binary", label: "Block binary" },
  { value: "block_certificate", label: "Block certificate" },
] as const;

export function SantaEventsPage() {
  const search = useSearch({ strict: false });
  const navigate = useNavigate();
  const [hostID, setHostID] = useDebouncedSearchParam("host_id", { resetKeys: ["after"] });
  const looseSearch = search as Record<string, unknown>;
  const decision =
    typeof looseSearch.decision === "string" && looseSearch.decision !== "" ? looseSearch.decision : "all";
  const after = typeof looseSearch.after === "string" ? looseSearch.after : undefined;
  const parsedHostID = Number(hostID);
  const query = useSantaEvents({
    host_id: hostID.trim() !== "" && Number.isFinite(parsedHostID) && parsedHostID > 0 ? parsedHostID : undefined,
    decision: decision === "all" ? undefined : decision,
    limit: 50,
    after,
  });
  const rows = query.data?.items ?? [];
  const nextCursor = query.data?.next_cursor;
  const hasFilters = !!hostID || decision !== "all";

  const columns: ColumnDef<SantaEvent>[] = [
    {
      id: "executable",
      header: "Executable",
      enableSorting: false,
      cell: ({ row }) => (
        <div className="grid gap-1">
          <span className="font-medium">{row.original.executable.file_name || row.original.executable.sha256}</span>
          <span className="text-muted-foreground max-w-[34rem] truncate text-xs">
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
      accessorKey: "host_id",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
      cell: ({ row }) => (
        <Link
          to="/hosts/$hostId"
          params={{ hostId: String(row.original.host_id) }}
          className="font-medium hover:underline"
        >
          {row.original.host_id}
        </Link>
      ),
    },
    {
      id: "executing_user",
      accessorKey: "executing_user",
      header: ({ column }) => <DataTableColumnHeader column={column} title="User" />,
      cell: ({ row }) => <span className="text-muted-foreground">{row.original.executing_user || "-"}</span>,
    },
    {
      id: "occurred_at",
      accessorFn: (row) => row.occurred_at ?? row.ingested_at,
      header: ({ column }) => <DataTableColumnHeader column={column} title="When" />,
      cell: ({ row }) => {
        const timestamp = row.original.occurred_at ?? row.original.ingested_at;
        return (
          <span className="text-muted-foreground" title={new Date(timestamp).toLocaleString()}>
            {formatRelative(timestamp)}
          </span>
        );
      },
    },
  ];

  function setDecision(next: string) {
    void navigate({
      search: ((prev: Record<string, unknown>) => ({
        ...prev,
        decision: next === "all" ? undefined : next,
        after: undefined,
      })) as never,
      replace: true,
    });
  }

  function setAfter(next: string | undefined) {
    void navigate({
      search: ((prev: Record<string, unknown>) => ({ ...prev, after: next })) as never,
      replace: true,
    });
  }

  return (
    <PageShell>
      <PageHeader title="Santa events" description="Recent execution events uploaded by Santa clients." />

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load Santa events</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          totalCount={rows.length}
          pagination={{ pageIndex: 0, pageSize: rows.length || 50 }}
          sorting={[]}
          onPaginationChange={() => undefined}
          onSortingChange={() => undefined}
          isLoading={query.isLoading}
          clientSort
          toolbar={
            <div className="flex flex-wrap items-end gap-2">
              <DataTableSearch value={hostID} onChange={setHostID} placeholder="Host ID" label="Filter by host ID" />
              <div className="grid gap-1">
                <Label className="text-muted-foreground text-xs">Decision</Label>
                <Select value={decision} onValueChange={setDecision}>
                  <SelectTrigger className="w-48">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {DECISION_FILTERS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          }
          empty={
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Activity />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matches" : "No events yet"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters
                    ? "No Santa execution events matched these filters."
                    : "Santa execution events will appear here after clients upload them."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          }
        />
      )}

      <div className="flex items-center gap-2">
        {after ? (
          <Button type="button" variant="outline" size="sm" onClick={() => setAfter(undefined)}>
            Newest
          </Button>
        ) : null}
        {nextCursor ? (
          <Button type="button" variant="outline" size="sm" onClick={() => setAfter(nextCursor)}>
            Next page
          </Button>
        ) : null}
      </div>
    </PageShell>
  );
}

function DecisionBadge({ decision }: { decision: string }) {
  const blocked = decision.startsWith("block_");
  const allowed = decision.startsWith("allow_");
  return (
    <Badge variant={blocked ? "destructive" : allowed ? "secondary" : "outline"}>{decision.replaceAll("_", " ")}</Badge>
  );
}
