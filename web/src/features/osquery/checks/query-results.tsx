import type { ColumnDef } from "@tanstack/react-table";

import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { Link } from "@components/link";
import type { OsqueryCheckHostStatus } from "@lib/api";
import { formatRelative } from "@lib/utils";

import { CHECK_RESULT_STATUSES, type CheckResultDisplayStatus } from "./model";

export type CheckResultRow = {
  host_id: number;
  host_name: string;
  status: CheckResultDisplayStatus;
  updated_at?: string;
  error?: string;
};

export function checkResultFromStatus(result: OsqueryCheckHostStatus): CheckResultRow {
  return {
    host_id: result.host_id,
    host_name: result.host_name,
    status: result.status,
    updated_at: result.updated_at,
  };
}

export function createCheckResultColumns({
  timestampHeader,
  includeError = false,
}: {
  timestampHeader: "Last Evaluated";
  includeError?: boolean;
}): ColumnDef<CheckResultRow>[] {
  const columns: ColumnDef<CheckResultRow>[] = [
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
      accessorKey: "status",
      header: () => "Status",
      enableColumnFilter: true,
      cell: ({ row }) => (
        <EnumStatusIndicator value={row.original.status} metadata={CHECK_RESULT_STATUSES} />
      ),
    },
    {
      accessorKey: "updated_at",
      header: () => timestampHeader,
      cell: ({ row }) => (row.original.updated_at ? formatRelative(row.original.updated_at) : "-"),
    },
  ];
  if (includeError) {
    columns.push({
      accessorKey: "error",
      header: () => "Error",
      cell: ({ row }) => row.original.error || "-",
    });
  }
  return columns;
}
