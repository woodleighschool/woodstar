import type { ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";

import { DataTableStatic } from "@components/data-table/data-table-static";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { useHostOsqueryChecks } from "@features/hosts/queries";
import { CHECK_RESULT_STATUSES, checkResultStatus } from "@features/osquery/checks/model";
import type { OsqueryCheckHostStatus } from "@lib/api";
import { formatRelative } from "@lib/utils";

const checkColumns: ColumnDef<OsqueryCheckHostStatus>[] = [
  {
    accessorKey: "check_name",
    header: () => "Check",
    cell: ({ row }) => (
      <Link to="/osquery/checks/$id" params={{ id: String(row.original.check_id) }}>
        {row.original.check_name}
      </Link>
    ),
  },
  {
    accessorKey: "response",
    header: () => "Status",
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
  if (rows.length === 0) return <PanelEmptyState>No checks yet</PanelEmptyState>;

  return <DataTableStatic columns={checkColumns} data={rows} />;
}
