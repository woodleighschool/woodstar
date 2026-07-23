import { Link, useParams, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { EmptyPanel } from "@/components/empty-panel";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { CheckStatusBadge } from "@/components/osquery/checks/check-status-badge";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCheck, useCheckResults } from "@/hooks/use-checks";
import type { OsqueryCheckHostStatus } from "@/lib/api";
import { formatRelative } from "@/lib/utils";
const resultColumns: ColumnDef<OsqueryCheckHostStatus>[] = [
  {
    accessorKey: "host_name",
    header: () => "Host",
    cell: ({ row }) => (
      <Link
        to="/hosts/$id"
        params={{ id: String(row.original.host_id) }}
        className="font-medium hover:underline"
      >
        {row.original.host_name}
      </Link>
    ),
  },
  {
    accessorKey: "response",
    header: () => "Status",
    cell: ({ row }) => <CheckStatusBadge response={row.original.response} />,
  },
  {
    accessorKey: "updated_at",
    header: () => "Last Evaluated",
    cell: ({ row }) => formatRelative(row.original.updated_at),
  },
];
export function CheckResultsPage() {
  const { id: checkId } = useParams({ from: "/_authenticated/osquery/checks/$id" });
  const search = useSearch({ from: "/_authenticated/osquery/checks/$id/results" });
  const id = Number(checkId);
  const check = useCheck(id);
  const results = useCheckResults(id, { response: search.response });
  const rows = results.data ?? [];
  const responseLabel =
    search.response === "pass" ? "Passing" : search.response === "fail" ? "Failing" : "All";
  return (
    <PageShell>
      <PageHeader
        title="Check Results"
        description={`${responseLabel} check results by host.`}
        actions={
          <Button
            variant="outline"
            size="sm"
            render={<Link to="/osquery/checks/$id" params={{ id: checkId }} />}
            nativeButton={false}
          >
            Edit Check
          </Button>
        }
      />

      {check.error ? (
        <QueryError
          title="Failed to load check"
          error={check.error}
          onRetry={() => void check.refetch()}
        />
      ) : results.error ? (
        <QueryError
          title="Failed to load check results"
          error={results.error}
          onRetry={() => void results.refetch()}
        />
      ) : check.isLoading || results.isLoading ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      ) : (
        <DataTableStatic
          columns={resultColumns}
          data={rows}
          empty={<EmptyPanel>No check results yet</EmptyPanel>}
        />
      )}
    </PageShell>
  );
}
