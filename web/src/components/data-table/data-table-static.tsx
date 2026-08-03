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

import {
  DATA_TABLE_DEFAULT_COLUMN,
  DataTableCellContent,
  DataTableResizeHandle,
  getDataTableColumnStyle,
} from "@components/data-table/data-table-sizing";
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

interface DataTableStaticProps<TData> extends React.ComponentProps<"section"> {
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
    defaultColumn: DATA_TABLE_DEFAULT_COLUMN,
    enableColumnResizing: true,
    columnResizeMode: "onChange",
  });

  return (
    <TableSurface heading={heading} className={cn(className)} {...props}>
      <Table className="block" style={{ width: `max(100%, ${table.getTotalSize()}px)` }}>
        <TableHeader className="block">
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id} className="flex w-full">
              {headerGroup.headers.map((header) => (
                <TableHead
                  key={header.id}
                  colSpan={header.colSpan}
                  style={getDataTableColumnStyle(header.column)}
                  className="group/table-head relative flex min-w-0 items-center overflow-visible"
                >
                  {header.isPlaceholder ? null : (
                    <div className="w-full min-w-0 truncate">
                      {flexRender(header.column.columnDef.header, header.getContext())}
                    </div>
                  )}
                  <DataTableResizeHandle header={header} />
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody className="block">
          {table.getRowModel().rows.length ? (
            table.getRowModel().rows.map((row) => (
              <React.Fragment key={row.id}>
                <TableRow className="group/row flex w-full">
                  {row.getVisibleCells().map((cell) => (
                    <TableCell
                      key={cell.id}
                      style={getDataTableColumnStyle(cell.column)}
                      className="flex min-w-0 items-center overflow-hidden"
                    >
                      <DataTableCellContent>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </DataTableCellContent>
                    </TableCell>
                  ))}
                </TableRow>
                {row.getCanExpand() && row.getIsExpanded() && renderSubRow ? (
                  <TableRow className="flex w-full hover:bg-transparent">
                    <TableCell
                      colSpan={row.getVisibleCells().length}
                      className="w-full max-w-none flex-1 border-b bg-muted/20 p-0"
                    >
                      {renderSubRow(row)}
                    </TableCell>
                  </TableRow>
                ) : null}
              </React.Fragment>
            ))
          ) : (
            <TableRow className="flex w-full hover:bg-transparent">
              <TableCell colSpan={columns.length} className="w-full max-w-none flex-1 p-0">
                {empty}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </TableSurface>
  );
}
