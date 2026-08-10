import type { CSSProperties, ReactNode } from "react";

import type {
  DataTableColumn,
  DataTableHeader,
  DataTableRowData,
} from "@components/data-table/types";
import { cn } from "@lib/utils";

// `size` is both the preferred flex basis and the relative share of spare room.
// Columns preserve their use-case minimum and only overflow the surface once
// their bases no longer fit.
export const DATA_TABLE_DEFAULT_COLUMN = {
  size: 160,
  minSize: 72,
  maxSize: Number.MAX_SAFE_INTEGER,
} as const;

export const DATA_TABLE_CONTROL_COLUMN = {
  size: 44,
  minSize: 44,
  maxSize: 44,
  enableResizing: false,
} as const;

export function getDataTableColumnStyle<TData extends DataTableRowData, TValue>(
  column: DataTableColumn<TData, TValue>,
): CSSProperties {
  const size = column.getSize();
  const minSize = column.columnDef.minSize ?? DATA_TABLE_DEFAULT_COLUMN.minSize;
  const maxSize = column.columnDef.maxSize ?? DATA_TABLE_DEFAULT_COLUMN.maxSize;
  const isFixed = minSize === maxSize;
  const isTrailingActions = column.id === "actions" && column.getAfter() === 0;

  return {
    flex: `${isFixed ? 0 : size / DATA_TABLE_DEFAULT_COLUMN.size} 0 ${size}px`,
    marginLeft: isTrailingActions ? "auto" : undefined,
    minWidth: minSize,
    maxWidth: maxSize === Number.MAX_SAFE_INTEGER ? undefined : maxSize,
    width: size,
  };
}

export function DataTableCellContent({ children }: { children: ReactNode }) {
  const title =
    typeof children === "string" || typeof children === "number" ? String(children) : undefined;

  return (
    <div className="w-full min-w-0 truncate" title={title}>
      {children}
    </div>
  );
}

export function DataTableResizeHandle<TData extends DataTableRowData, TValue>({
  header,
}: {
  header: DataTableHeader<TData, TValue>;
}) {
  if (!header.column.getCanResize()) return null;

  return (
    <button
      type="button"
      data-slot="data-table-resize-handle"
      aria-label="Resize column"
      tabIndex={-1}
      onDoubleClick={() => header.column.resetSize()}
      onMouseDown={header.getResizeHandler()}
      onTouchStart={header.getResizeHandler()}
      className="absolute inset-y-0 right-0 z-10 w-2 translate-x-1/2 cursor-col-resize touch-none select-none"
    >
      <span
        className={cn(
          "absolute inset-y-1 left-1/2 w-px -translate-x-1/2 bg-border opacity-0 transition-opacity group-hover/table-head:opacity-100 hover:opacity-100",
          header.column.getIsResizing() && "bg-primary opacity-100",
        )}
      />
    </button>
  );
}
