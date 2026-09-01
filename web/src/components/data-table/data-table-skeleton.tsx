import * as React from "react";

import { TableSurface } from "@components/data-table/table-surface";
import { Skeleton } from "@components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";

interface DataTableSkeletonProps extends React.ComponentProps<"div"> {
  columnCount: number;
  rowCount?: number;
  filterCount?: number;
  withExport?: boolean;
  withViewOptions?: boolean;
  delayMs?: number;
}

export function DataTableSkeleton({
  columnCount,
  rowCount = 10,
  filterCount = 0,
  withExport = false,
  withViewOptions = false,
  delayMs = 150,
  className,
  ...props
}: DataTableSkeletonProps) {
  const [delayElapsed, setDelayElapsed] = React.useState(false);

  React.useEffect(() => {
    if (delayMs <= 0) return undefined;

    const timer = window.setTimeout(() => setDelayElapsed(true), delayMs);
    return () => window.clearTimeout(timer);
  }, [delayMs]);

  if (delayMs > 0 && !delayElapsed) return null;

  return (
    <TableSurface
      className={className}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2 p-1">
          <Skeleton className="h-8 max-w-sm min-w-48 flex-1 basis-48" />
          {Array.from({ length: filterCount }).map((_, i) => (
            <Skeleton key={i} className="h-8 w-24" />
          ))}
          {withViewOptions || withExport ? (
            <div className="ml-auto flex shrink-0 items-center gap-2">
              {withViewOptions ? <Skeleton className="h-7 w-18" /> : null}
              {withExport ? <Skeleton className="h-7 w-18" /> : null}
            </div>
          ) : null}
        </div>
      }
      footer={
        <div className="flex w-full flex-wrap items-center justify-between gap-3 border-t p-3">
          <Skeleton className="h-7 w-32" />
          <div className="flex flex-wrap items-center justify-end gap-3">
            <div className="flex items-center gap-2">
              <Skeleton className="h-7 w-24" />
              <Skeleton className="h-8 w-18" />
            </div>
            <Skeleton className="h-7 w-20" />
            <Skeleton className="h-8 w-32" />
          </div>
        </div>
      }
      {...props}
    >
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            {Array.from({ length: columnCount }).map((_column, columnIndex) => (
              <TableHead key={columnIndex}>
                <Skeleton className="h-6 w-full min-w-20" />
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {Array.from({ length: rowCount }).map((_row, rowIndex) => (
            <TableRow key={rowIndex} className="hover:bg-transparent">
              {Array.from({ length: columnCount }).map((_column, columnIndex) => (
                <TableCell key={columnIndex}>
                  <Skeleton className="h-6 w-full min-w-20" />
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableSurface>
  );
}
