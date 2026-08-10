import { type CSSProperties, type ReactNode, useCallback, useRef, useState } from "react";

import type {
  DataTableColumn,
  DataTableHeader,
  DataTableRowData,
} from "@components/data-table/types";
import { Tooltip, TooltipContent, TooltipTrigger } from "@components/ui/tooltip";
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
  const contentRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [tooltipText, setTooltipText] = useState("");

  const handleOpenChange = useCallback((nextOpen: boolean) => {
    if (!nextOpen) {
      setOpen(false);
      return;
    }

    const content = contentRef.current;
    if (!content || !hasOverflow(content)) return;

    const text = content.innerText.trim();
    if (!text) return;

    setTooltipText(text);
    setOpen(true);
  }, []);

  return (
    <Tooltip open={open} onOpenChange={handleOpenChange}>
      <TooltipTrigger render={<div ref={contentRef} className="w-full min-w-0 truncate" />}>
        {children}
      </TooltipTrigger>
      <TooltipContent className="max-w-[calc(100vw-2rem)] break-all whitespace-normal sm:max-w-lg">
        {tooltipText}
      </TooltipContent>
    </Tooltip>
  );
}

function hasOverflow(root: HTMLElement) {
  return [root, ...root.querySelectorAll("*")].some(
    (element) => element.scrollWidth > element.clientWidth,
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
