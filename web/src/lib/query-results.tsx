import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";

import { DataTableColumnHeader } from "@/components/ui/data-table-column-header";
import type { Schemas } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

export type QueryResultRow = Schemas["QueryResultBody"];

export type ReportTableRow = {
  id: string;
  queryId: number;
  hostId: number;
  hostName: string;
  lastFetched?: string;
  columns: Record<string, string>;
};

export function reportRows(rows: QueryResultRow[] | null | undefined): ReportTableRow[] {
  return (rows ?? []).map((row, index) => ({
    id: `${row.query_id}-${row.host_id}-${index}`,
    queryId: row.query_id,
    hostId: row.host_id,
    hostName: row.host_name || String(row.host_id),
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
  return Array.from(seen).sort((a, b) => a.localeCompare(b));
}

export function reportTableColumns(options: { linkHosts?: boolean } = {}): ColumnDef<ReportTableRow>[] {
  return [
    {
      id: "hostName",
      accessorKey: "hostName",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
      cell: ({ row }) =>
        options.linkHosts ? (
          <Link
            to="/hosts/$hostId"
            params={{ hostId: String(row.original.hostId) }}
            className="font-medium hover:underline"
          >
            {row.original.hostName}
          </Link>
        ) : (
          <span className="font-medium">{row.original.hostName}</span>
        ),
    },
    {
      id: "lastFetched",
      accessorKey: "lastFetched",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Last fetched" />,
      cell: ({ row }) => (
        <span
          className="text-muted-foreground"
          title={row.original.lastFetched ? new Date(row.original.lastFetched).toLocaleString() : ""}
        >
          {row.original.lastFetched ? formatRelative(row.original.lastFetched) : "-"}
        </span>
      ),
    },
  ];
}

export function resultValue(value: string | undefined) {
  if (value == null || value === "") return "-";
  return value;
}
