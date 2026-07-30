import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  getExpandedRowModel,
  type Row,
  type TableOptions,
  useReactTable,
} from "@tanstack/react-table";
import * as React from "react";

import { TableSurface } from "@components/data-table/table-surface";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { cn } from "@lib/utils";

interface DataTableStaticProps<TData> extends React.ComponentProps<"div"> {
  columns: ColumnDef<TData>[];
  data: TData[];
  empty?: React.ReactNode;
  heading?: React.ReactNode;
  getRowCanExpand?: TableOptions<TData>["getRowCanExpand"];
  getRowId?: TableOptions<TData>["getRowId"];
  renderSubRow?: (row: Row<TData>) => React.ReactNode;
}

// Presentational table for nested/local row sets (no pagination, no URL state).
// Use this for detail-page tables and pickers; the server DataTable is only for
// top-level paginated lists.
export function DataTableStatic<TData>({
  columns,
  data,
  empty,
  heading,
  getRowCanExpand,
  getRowId,
  renderSubRow,
  className,
  ...props
}: DataTableStaticProps<TData>) {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
    getRowCanExpand,
    getRowId,
  });

  return (
    <TableSurface
      heading={heading}
      className={cn(className)}
      viewportClassName="max-h-96"
      {...props}
    >
      <Table>
        <TableHeader className="sticky top-0 z-10">
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <TableHead key={header.id} colSpan={header.colSpan}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows.length ? (
            table.getRowModel().rows.map((row) => (
              <React.Fragment key={row.id}>
                <TableRow className="group/row">
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
                {row.getCanExpand() && row.getIsExpanded() && renderSubRow ? (
                  <TableRow className="hover:bg-transparent">
                    <TableCell
                      colSpan={row.getVisibleCells().length}
                      className="border-b bg-muted/20 p-0"
                    >
                      {renderSubRow(row)}
                    </TableCell>
                  </TableRow>
                ) : null}
              </React.Fragment>
            ))
          ) : (
            <TableRow className="hover:bg-transparent">
              <TableCell colSpan={columns.length} className="p-0">
                {empty}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </TableSurface>
  );
}
