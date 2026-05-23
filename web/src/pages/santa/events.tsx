import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import { Activity, Loader2 } from "lucide-react";

import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
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
  const query = useSantaEvents({
    host_id: hostID,
    decision: decision === "all" ? undefined : decision,
    limit: 50,
    after,
  });
  const rows = query.data?.items ?? [];
  const nextCursor = query.data?.next_cursor;

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

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load Santa events</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : (
        <SantaEventTable rows={rows} loading={query.isLoading} />
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

function SantaEventTable({ rows, loading }: { rows: SantaEvent[]; loading: boolean }) {
  if (loading && rows.length === 0) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading...
      </div>
    );
  }

  if (rows.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Activity />
          </EmptyMedia>
          <EmptyTitle>No events</EmptyTitle>
          <EmptyDescription>No Santa execution events matched these filters.</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Executable</TableHead>
            <TableHead>Decision</TableHead>
            <TableHead>Host</TableHead>
            <TableHead>User</TableHead>
            <TableHead>When</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((event) => (
            <TableRow key={event.id}>
              <TableCell className="min-w-[18rem]">
                <div className="grid gap-1">
                  <span className="font-medium">{event.executable.file_name || event.executable.sha256}</span>
                  <span className="text-muted-foreground max-w-[34rem] truncate text-xs">
                    {event.file_path || event.executable.sha256}
                  </span>
                </div>
              </TableCell>
              <TableCell>
                <DecisionBadge decision={event.decision} />
              </TableCell>
              <TableCell>
                <Link
                  to="/hosts/$hostId"
                  params={{ hostId: String(event.host_id) }}
                  className="font-medium hover:underline"
                >
                  {event.host_id}
                </Link>
              </TableCell>
              <TableCell className="text-muted-foreground">{event.executing_user || "-"}</TableCell>
              <TableCell
                className="text-muted-foreground"
                title={new Date(event.occurred_at ?? event.ingested_at).toLocaleString()}
              >
                {formatRelative(event.occurred_at ?? event.ingested_at)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function DecisionBadge({ decision }: { decision: string }) {
  const blocked = decision.startsWith("block_");
  const allowed = decision.startsWith("allow_");
  return (
    <Badge variant={blocked ? "destructive" : allowed ? "secondary" : "outline"}>{decision.replaceAll("_", " ")}</Badge>
  );
}
