import type { ColumnDef } from "@tanstack/react-table";

import { DATA_TABLE_CONTROL_COLUMN } from "@components/data-table/data-table-sizing";
import { Checkbox } from "@components/ui/checkbox";

// Row-selection column shared by resource lists that support bulk actions.
export function selectColumn<TData>(): ColumnDef<TData> {
  return {
    ...DATA_TABLE_CONTROL_COLUMN,
    id: "select",
    header: ({ table }) => (
      <div className="flex w-full justify-center">
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          indeterminate={!table.getIsAllPageRowsSelected() && table.getIsSomePageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(value)}
        />
      </div>
    ),
    cell: ({ row }) => (
      <div className="flex w-full justify-center">
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(value)}
        />
      </div>
    ),
    enableSorting: false,
    enableHiding: false,
  };
}
