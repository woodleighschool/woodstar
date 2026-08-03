import { flexRender, type Row, type Table as TanstackTable } from "@tanstack/react-table";
import { ArrowDownIcon, ArrowUpDownIcon, ArrowUpIcon } from "lucide-react";
import * as React from "react";

import {
  DataTableExport,
  type DataTableExportOptions,
} from "@components/data-table/data-table-export";
import { DataTablePagination } from "@components/data-table/data-table-pagination";
import {
  DataTableCellContent,
  DataTableResizeHandle,
  getDataTableColumnStyle,
} from "@components/data-table/data-table-sizing";
import { TableSurface } from "@components/data-table/table-surface";
import { Button } from "@components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { cn } from "@lib/utils";

interface DataTableProps<TData> extends React.ComponentProps<"div"> {
  table: TanstackTable<TData>;
  actionBar?: React.ReactNode;
  empty?: React.ReactNode;
  exportOptions?: DataTableExportOptions<TData>;
  heading?: React.ReactNode;
  pageSizeOptions?: readonly number[];
  renderSubRow?: (row: Row<TData>) => React.ReactNode;
  toolbarActions?: React.ReactNode;
}

export function DataTable<TData>({
  table,
  actionBar,
  empty,
  exportOptions,
  heading,
  pageSizeOptions,
  renderSubRow,
  toolbarActions,
  children,
  className,
  ...props
}: DataTableProps<TData>) {
  const toolbar =
    children || toolbarActions || exportOptions ? (
      <div className="flex flex-wrap items-center gap-2 p-1">
        {children ? (
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">{children}</div>
        ) : (
          <div className="flex-1" />
        )}
        <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
          {toolbarActions}
          {exportOptions ? <DataTableExport table={table} options={exportOptions} /> : null}
        </div>
      </div>
    ) : null;

  return (
    <div className={cn("min-w-0", className)} {...props}>
      <TableSurface
        heading={heading}
        toolbar={toolbar}
        footer={
          <DataTablePagination
            table={table}
            pageSizeOptions={pageSizeOptions}
            className="border-t px-3 py-2"
          />
        }
      >
        <Table className="block" style={{ width: `max(100%, ${table.getTotalSize()}px)` }}>
          <TableHeader className="block">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id} className="flex w-full">
                {headerGroup.headers.map((header) => {
                  const direction = header.column.getIsSorted();
                  const label = flexRender(header.column.columnDef.header, header.getContext());
                  const SortIcon =
                    direction === "asc"
                      ? ArrowUpIcon
                      : direction === "desc"
                        ? ArrowDownIcon
                        : ArrowUpDownIcon;

                  return (
                    <TableHead
                      key={header.id}
                      colSpan={header.colSpan}
                      style={getDataTableColumnStyle(header.column)}
                      className="group/table-head relative flex min-w-0 items-center overflow-visible"
                      aria-sort={
                        direction === "asc"
                          ? "ascending"
                          : direction === "desc"
                            ? "descending"
                            : "none"
                      }
                    >
                      {header.isPlaceholder ? null : header.column.getCanSort() ? (
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="-ml-2 h-8 max-w-full justify-start overflow-hidden font-medium text-ellipsis"
                          onClick={header.column.getToggleSortingHandler()}
                        >
                          <span className="min-w-0 truncate">{label}</span>
                          <SortIcon
                            data-icon="inline-end"
                            className={cn(!direction && "text-muted-foreground")}
                          />
                        </Button>
                      ) : (
                        <div className="w-full min-w-0 truncate">{label}</div>
                      )}
                      <DataTableResizeHandle header={header} />
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody className="block">
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <React.Fragment key={row.id}>
                  <TableRow
                    className="group/row flex w-full"
                    data-state={row.getIsSelected() && "selected"}
                  >
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
                <TableCell
                  colSpan={table.getVisibleLeafColumns().length}
                  className="w-full max-w-none flex-1 p-0"
                >
                  {empty}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableSurface>
      {actionBar}
    </div>
  );
}
