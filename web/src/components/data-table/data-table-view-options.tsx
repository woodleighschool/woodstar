import { Settings2 } from "lucide-react";
import * as React from "react";

import type { DataTableInstance, DataTableRowData } from "@components/data-table/types";
import { Button } from "@components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@components/ui/dropdown-menu";

interface DataTableViewOptionsProps<TData extends DataTableRowData> extends React.ComponentProps<
  typeof DropdownMenuContent
> {
  table: DataTableInstance<TData>;
  disabled?: boolean;
}

export function DataTableViewOptions<TData extends DataTableRowData>({
  table,
  disabled,
  ...props
}: DataTableViewOptionsProps<TData>) {
  const columns = React.useMemo(
    () =>
      table
        .getAllColumns()
        .filter((column) => typeof column.accessorFn !== "undefined" && column.getCanHide()),
    [table],
  );

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button variant="outline" size="sm" className="h-8 font-normal" disabled={disabled} />
        }
      >
        <Settings2 data-icon="inline-start" className="text-muted-foreground" />
        View
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-44" {...props}>
        <DropdownMenuGroup>
          <DropdownMenuLabel>Toggle Columns</DropdownMenuLabel>
          {columns.map((column) => (
            <DropdownMenuCheckboxItem
              key={column.id}
              checked={column.getIsVisible()}
              onCheckedChange={(checked) => column.toggleVisibility(checked)}
            >
              <span>{column.columnDef.meta?.label ?? column.id}</span>
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
