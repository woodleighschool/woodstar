import { type ColumnDef, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import type * as React from "react";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface DataTableStaticProps<TData> {
  columns: ColumnDef<TData>[];
  data: TData[];
  empty?: React.ReactNode;
}

// Presentational table for nested/local row sets (no pagination, no URL state).
// Use this for detail-page tables and pickers; the server DataTable is only for
// top-level paginated lists.
export function DataTableStatic<TData>({ columns, data, empty }: DataTableStaticProps<TData>) {
  const table = useReactTable({ data, columns, getCoreRowModel: getCoreRowModel() });

  return (
    <div className="max-h-96 overflow-auto rounded-lg border bg-card *:data-[slot=table-container]:overflow-visible">
      <Table>
        <TableHeader className="sticky top-0 z-10 bg-muted">
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
        <TableBody className="[&_a:not([data-slot=button])]:underline [&_a:not([data-slot=button])]:decoration-dotted [&_a:not([data-slot=button])]:underline-offset-4 [&_a:not([data-slot=button]):focus-visible]:decoration-solid [&_a:not([data-slot=button]):hover]:decoration-solid [&_button:not([data-slot=button])]:underline [&_button:not([data-slot=button])]:decoration-dotted [&_button:not([data-slot=button])]:underline-offset-4 [&_button:not([data-slot=button]):focus-visible]:decoration-solid [&_button:not([data-slot=button]):hover]:decoration-solid">
          {table.getRowModel().rows.length ? (
            table.getRowModel().rows.map((row) => (
              <TableRow key={row.id}>
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
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
    </div>
  );
}
