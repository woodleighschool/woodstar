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
import type {
  DataTableInstance,
  DataTableRow,
  DataTableRowData,
} from "@components/data-table/types";
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

interface DataTableProps<TData extends DataTableRowData> extends React.ComponentProps<"div"> {
  table: DataTableInstance<TData>;
  actionBar?: React.ReactNode;
  empty?: React.ReactNode;
  exportOptions?: DataTableExportOptions<TData>;
  heading?: React.ReactNode;
  pageSizeOptions?: readonly number[];
  pending?: boolean;
  renderSubRow?: (row: DataTableRow<TData>) => React.ReactNode;
  toolbarActions?: React.ReactNode;
}

export function DataTable<TData extends DataTableRowData>({
  table,
  actionBar,
  empty,
  exportOptions,
  heading,
  pageSizeOptions,
  pending = false,
  renderSubRow,
  toolbarActions,
  children,
  className,
  ...props
}: DataTableProps<TData>) {
  const toolbarControls = React.Children.toArray(children);
  const [primaryToolbarControl, ...secondaryToolbarControls] = toolbarControls;
  const hasToolbarActions = Boolean(toolbarActions || exportOptions);
  const toolbar =
    toolbarControls.length || hasToolbarActions ? (
      <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 p-1 sm:flex sm:flex-wrap">
        {primaryToolbarControl ? (
          <div className="col-span-2 min-w-0 *:max-w-none sm:contents sm:*:max-w-sm">
            {primaryToolbarControl}
          </div>
        ) : null}
        {secondaryToolbarControls.length ? (
          <div
            className={cn(
              "flex min-w-0 flex-wrap items-center gap-2 sm:contents",
              !hasToolbarActions && "col-span-2",
            )}
          >
            {secondaryToolbarControls}
          </div>
        ) : null}
        {hasToolbarActions ? (
          <div
            className={cn(
              "ml-auto flex shrink-0 items-center gap-2",
              primaryToolbarControl
                ? "col-start-2 row-start-2 sm:col-auto sm:row-auto"
                : "col-span-2 justify-self-end",
            )}
          >
            {toolbarActions}
            {exportOptions ? <DataTableExport table={table} options={exportOptions} /> : null}
          </div>
        ) : null}
      </div>
    ) : null;

  return (
    <div className={cn("min-w-0", className)} {...props} aria-busy={pending || undefined}>
      <TableSurface
        heading={heading}
        toolbar={toolbar}
        empty={table.getRowModel().rows.length ? undefined : empty}
        footer={
          <DataTablePagination
            table={table}
            pageSizeOptions={pageSizeOptions}
            className="border-t px-3 py-2"
          />
        }
      >
        <Table
          className={cn(
            "block transition-opacity duration-150",
            pending ? "opacity-60 delay-150" : "opacity-100 delay-0",
          )}
          style={{ width: `max(100%, ${table.getTotalSize()}px)` }}
        >
          <TableHeader className="block">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id} className="flex w-full">
                {headerGroup.headers.map((header) => {
                  const direction = header.column.getIsSorted();
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
                          <span className="min-w-0 truncate">
                            <table.FlexRender header={header} />
                          </span>
                          <SortIcon
                            data-icon="inline-end"
                            className={cn(!direction && "text-muted-foreground")}
                          />
                        </Button>
                      ) : (
                        <div className="w-full min-w-0 truncate">
                          <table.FlexRender header={header} />
                        </div>
                      )}
                      <DataTableResizeHandle header={header} />
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody className="block">
            {table.getRowModel().rows.length
              ? table.getRowModel().rows.map((row) => (
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
                            <table.FlexRender cell={cell} />
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
              : null}
          </TableBody>
        </Table>
      </TableSurface>
      {actionBar}
    </div>
  );
}
