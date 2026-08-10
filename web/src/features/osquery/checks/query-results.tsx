import type { DataTableColumnDef } from "@components/data-table/types";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { TextLink } from "@components/link";
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
}): DataTableColumnDef<CheckResultRow>[] {
  const columns: DataTableColumnDef<CheckResultRow>[] = [
    {
      accessorKey: "host_name",
      header: () => "Host",
      cell: ({ row }) => (
        <TextLink
          to="/hosts/$id"
          params={{ id: String(row.original.host_id) }}
          className="font-medium"
        >
          {row.original.host_name}
        </TextLink>
      ),
    },
    {
      accessorKey: "status",
      header: () => "Status",
      enableColumnFilter: true,
      cell: ({ row }) => <CheckResultStatus status={row.original.status} />,
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

export function checkResultStatusLabel(status: CheckResultDisplayStatus) {
  return CHECK_RESULT_STATUSES[status].name;
}

export function CheckResultStatus({ status }: { status: CheckResultDisplayStatus }) {
  return <EnumStatusIndicator value={status} metadata={CHECK_RESULT_STATUSES} />;
}
