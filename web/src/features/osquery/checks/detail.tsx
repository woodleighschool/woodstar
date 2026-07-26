import { useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef, Table } from "@tanstack/react-table";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { DataTableClient } from "@components/data-table/data-table-client";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { KeyValueGrid, KeyValueItem } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { Card, CardContent } from "@components/ui/card";
import { Skeleton } from "@components/ui/skeleton";
import { useAuth } from "@features/auth/queries";
import { CHECK_RESULT_STATUSES, checkResultStatus } from "@features/osquery/checks/model";
import { LiveRunButton, ShowQueryButton } from "@features/osquery/live/query-actions";
import type { OsqueryCheckHostStatus } from "@lib/api";
import { parseRouteID } from "@lib/route-params";
import { formatRelative } from "@lib/utils";

import { CheckDeleteDialog } from "./delete-dialog";
import { useCheck, useCheckResults } from "./queries";

const resultColumns: ColumnDef<OsqueryCheckHostStatus>[] = [
  {
    accessorKey: "host_name",
    header: () => "Host",
    cell: ({ row }) => (
      <Link to="/hosts/$id" params={{ id: String(row.original.host_id) }} className="font-medium">
        {row.original.host_name}
      </Link>
    ),
  },
  {
    accessorKey: "response",
    header: () => "Status",
    filterFn: (row, id, value: string[]) => value.includes(row.getValue(id)),
    cell: ({ row }) => (
      <EnumStatusIndicator
        value={checkResultStatus(row.original.response)}
        metadata={CHECK_RESULT_STATUSES}
      />
    ),
  },
  {
    accessorKey: "updated_at",
    header: () => "Last Evaluated",
    cell: ({ row }) => formatRelative(row.original.updated_at),
  },
];

function CheckResultsToolbar({ table }: { table: Table<OsqueryCheckHostStatus> }) {
  return (
    <DataTableFacetedFilter
      column={table.getColumn("response")}
      title="Status"
      options={[
        { label: "Passing", value: "pass" },
        { label: "Failing", value: "fail" },
      ]}
    />
  );
}

function renderCheckResultsToolbar(table: Table<OsqueryCheckHostStatus>) {
  return <CheckResultsToolbar table={table} />;
}

export function CheckDetailPage() {
  const { id: checkId } = useParams({
    from: "/_authenticated/osquery/checks/$id",
  });
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deleteOpen, setDeleteOpen] = useState(false);
  const id = parseRouteID(checkId);
  const check = useCheck(id);
  const results = useCheckResults(id);
  const rows = results.data ?? [];

  if (id === null) {
    return (
      <QueryGate title="Failed to load check" error={{ message: "Check route is invalid." }} />
    );
  }

  if (check.error) {
    return (
      <QueryGate
        title="Failed to load check"
        error={check.error}
        onRetry={() => void check.refetch()}
      />
    );
  }

  if (!check.data) {
    return (
      <PageShell>
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </PageShell>
    );
  }

  return (
    <PageShell>
      <PageHeader
        title="Check Details"
        description={check.data.description || undefined}
        actions={
          <>
            {isAdmin ? (
              <>
                <Button
                  size="sm"
                  render={<Link to="/osquery/checks/$id/edit" params={{ id: checkId }} />}
                  nativeButton={false}
                >
                  <Pencil data-icon="inline-start" />
                  Edit
                </Button>
                <Button
                  type="button"
                  variant="destructive"
                  size="sm"
                  onClick={() => setDeleteOpen(true)}
                >
                  <Trash2 data-icon="inline-start" />
                  Delete
                </Button>
              </>
            ) : null}
            <ShowQueryButton sql={check.data.query} />
            <LiveRunButton to="/osquery/checks/$id/live" params={{ id: checkId }} />
          </>
        }
      />

      <Card>
        <CardContent>
          <KeyValueGrid>
            <KeyValueItem label="Name" value={check.data.name} />
            <KeyValueItem label="Passing" value={formatHostCount(check.data.passing_host_count)} />
            <KeyValueItem label="Failing" value={formatHostCount(check.data.failing_host_count)} />
            <KeyValueItem label="Updated" value={formatRelative(check.data.updated_at)} />
          </KeyValueGrid>
        </CardContent>
      </Card>

      {results.error ? (
        <QueryError
          title="Failed to load check results"
          error={results.error}
          onRetry={() => void results.refetch()}
        />
      ) : results.isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <DataTableClient
          columns={resultColumns}
          data={rows}
          initialSorting={[{ id: "host_name", desc: false }]}
          searchPlaceholder="Search check results"
          toolbar={renderCheckResultsToolbar}
          empty={<PanelEmptyState>No check results yet</PanelEmptyState>}
        />
      )}

      {isAdmin ? (
        <CheckDeleteDialog
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
          check={check.data}
          onDeleted={() => void navigate({ to: "/osquery/checks" })}
        />
      ) : null}
    </PageShell>
  );
}

function formatHostCount(count: number) {
  return `${count} ${count === 1 ? "host" : "hosts"}`;
}
