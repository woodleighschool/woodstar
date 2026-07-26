import type { ColumnDef } from "@tanstack/react-table";

import { Link } from "@components/link";
import type { OsqueryReportResult } from "@lib/api";
import { formatRelative } from "@lib/utils";

export type ReportTableRow = {
  id: string;
  hostId: number;
  hostName: string;
  lastFetched?: string;
  columns: Record<string, string>;
};

export function reportRows(rows: OsqueryReportResult[] | null | undefined): ReportTableRow[] {
  return (rows ?? []).map((row, index) => ({
    id: `${row.report_id}-${row.host_id}-${index}`,
    hostId: row.host_id,
    hostName: row.host_name,
    lastFetched: row.last_fetched,
    columns: row.columns,
  }));
}

export function resultColumnNames(rows: ReportTableRow[]): string[] {
  const seen = new Set<string>();
  for (const row of rows) {
    for (const key of Object.keys(row.columns)) {
      seen.add(key);
    }
  }
  return Array.from(seen).toSorted((a, b) => a.localeCompare(b));
}

export function reportTableColumns(): ColumnDef<ReportTableRow>[] {
  return [
    {
      id: "hostName",
      accessorKey: "hostName",
      header: () => "Host",
      cell: ({ row }) => (
        <Link to="/hosts/$id" params={{ id: String(row.original.hostId) }}>
          {row.original.hostName}
        </Link>
      ),
    },
    {
      id: "lastFetched",
      accessorKey: "lastFetched",
      header: () => "Last Fetched",
      cell: ({ row }) =>
        row.original.lastFetched ? formatRelative(row.original.lastFetched) : "-",
    },
  ];
}

export function resultValue(value: string | undefined) {
  if (value == null || value === "") return "-";
  return value;
}
