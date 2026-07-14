import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { EmptyPanel } from "@/components/empty-panel";
import { CheckStatusBadge } from "@/components/osquery/checks/check-status-badge";
import { QueryError } from "@/components/query-error";
import { useHostOsqueryChecks } from "@/hooks/use-hosts";
import type { OsqueryCheckHostStatus } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

const checkColumns: ColumnDef<OsqueryCheckHostStatus>[] = [
  {
    accessorKey: "check_name",
    header: () => "Check",
    cell: ({ row }) => (
      <Link
        to="/osquery/checks/$checkId"
        params={{ checkId: String(row.original.check_id) }}
        className="hover:underline"
      >
        {row.original.check_name}
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
    cell: ({ row }) => (row.original.updated_at ? formatRelative(row.original.updated_at) : "-"),
  },
];

export function HostOsqueryChecksTab({ hostId }: { hostId: number | null }) {
  const query = useHostOsqueryChecks(hostId);
  const rows = useMemo(
    () => [...(query.data ?? [])].toSorted((a, b) => a.check_name.localeCompare(b.check_name)),
    [query.data],
  );

  if (query.error) {
    return (
      <QueryError
        title="Failed to load checks"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  if (query.isLoading) return null;
  if (rows.length === 0) return <EmptyPanel>No checks yet</EmptyPanel>;

  return <DataTableStatic columns={checkColumns} data={rows} />;
}
