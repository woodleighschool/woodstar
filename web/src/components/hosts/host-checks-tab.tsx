import { Link } from "@tanstack/react-router";
import type { ColumnDef, PaginationState, SortingState } from "@tanstack/react-table";
import { ShieldCheck } from "lucide-react";
import { useMemo, useState } from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableEmptyState } from "@/components/data-table/data-table-empty-state";
import { CheckStatusBadge } from "@/components/osquery/checks/check-status-badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { useHostChecks } from "@/hooks/use-hosts";
import type { Schemas } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

type HostCheck = Schemas["CheckHostStatus"];

const HOST_CHECKS_PAGE_SIZE = 25;

export function HostChecksTab({ hostId }: { hostId: number | null }) {
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: HOST_CHECKS_PAGE_SIZE,
  });
  const [sorting, setSorting] = useState<SortingState>([{ id: "check_name", desc: false }]);
  const query = useHostChecks(hostId);
  const rows = query.data?.items ?? [];

  const columns = useMemo<ColumnDef<HostCheck>[]>(
    () => [
      {
        accessorKey: "check_name",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Check" />,
        cell: ({ row }) => (
          <Link
            to="/osquery/checks/$checkId"
            params={{ checkId: String(row.original.check_id) }}
            className="font-medium hover:underline"
          >
            {row.original.check_name || String(row.original.check_id)}
          </Link>
        ),
      },
      {
        accessorKey: "response",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
        cell: ({ row }) => <CheckStatusBadge response={row.original.response} />,
      },
      {
        accessorKey: "updated_at",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Last Evaluated" />,
        cell: ({ row }) => (row.original.updated_at ? formatRelative(row.original.updated_at) : "-"),
      },
    ],
    [],
  );

  if (query.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to Load Checks</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
      </Alert>
    );
  }

  return (
    <DataTable
      columns={columns}
      data={rows}
      totalCount={rows.length}
      pagination={pagination}
      sorting={sorting}
      onPaginationChange={setPagination}
      onSortingChange={setSorting}
      isLoading={query.isLoading}
      getRowId={(row) => `${row.check_id}-${row.host_id}`}
      rowHref={(row) => ({ to: "/osquery/checks/$checkId", params: { checkId: String(row.check_id) } })}
      empty={
        <DataTableEmptyState
          icon={<ShieldCheck />}
          title="No Checks"
          description="Add an osquery check to view pass/fail status for this host."
        />
      }
    />
  );
}
