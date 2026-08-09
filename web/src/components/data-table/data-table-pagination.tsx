import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from "lucide-react";

import type { DataTableInstance, DataTableRowData } from "@components/data-table/types";
import { Button } from "@components/ui/button";
import { ButtonGroup } from "@components/ui/button-group";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@components/ui/select";
import { PAGE_SIZE_OPTIONS } from "@lib/pagination";
import { cn } from "@lib/utils";

interface DataTablePaginationProps<
  TData extends DataTableRowData,
> extends React.ComponentProps<"div"> {
  table: DataTableInstance<TData>;
  pageSizeOptions?: readonly number[];
}

export function DataTablePagination<TData extends DataTableRowData>({
  table,
  pageSizeOptions = PAGE_SIZE_OPTIONS,
  className,
  ...props
}: DataTablePaginationProps<TData>) {
  const { pageIndex, pageSize } = table.state.pagination;
  const pageRowCount = table.getRowModel().rows.length;
  const rowCount = table.options.manualPagination
    ? table.getRowCount()
    : table.getPrePaginatedRowModel().rows.length;
  const firstRow = pageRowCount === 0 ? 0 : pageIndex * pageSize + 1;
  const lastRow = pageRowCount === 0 ? 0 : Math.min(rowCount, firstRow + pageRowCount - 1);

  return (
    <div
      className={cn(
        "grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-2 p-1 sm:flex sm:flex-wrap sm:justify-between sm:gap-3",
        className,
      )}
      {...props}
    >
      <div className="min-w-fit text-sm text-muted-foreground sm:flex-1">
        {firstRow}-{lastRow} of {rowCount}
      </div>
      <div className="flex items-center gap-2 justify-self-end">
        <p className="text-sm font-medium">Rows per page</p>
        <Select
          value={`${pageSize}`}
          onValueChange={(value) => {
            table.setPageSize(Number(value));
          }}
        >
          <SelectTrigger className="h-8 w-18 data-size:h-8">
            <SelectValue placeholder={pageSize} />
          </SelectTrigger>
          <SelectContent side="top">
            <SelectGroup>
              {pageSizeOptions.map((option) => (
                <SelectItem key={option} value={`${option}`}>
                  {option}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>
      <div className="col-span-2 flex items-center justify-between gap-3 sm:contents">
        <div className="text-sm font-medium">
          Page {pageIndex + 1} of {table.getPageCount()}
        </div>
        <ButtonGroup>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="hidden sm:inline-flex"
            onClick={() => table.setPageIndex(0)}
            disabled={!table.getCanPreviousPage()}
          >
            <ChevronsLeft />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
          >
            <ChevronLeft />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
          >
            <ChevronRight />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="hidden sm:inline-flex"
            onClick={() => table.setPageIndex(table.getPageCount() - 1)}
            disabled={!table.getCanNextPage()}
          >
            <ChevronsRight />
          </Button>
        </ButtonGroup>
      </div>
    </div>
  );
}
