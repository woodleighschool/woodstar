import type { ColumnDef } from "@tanstack/react-table";
import { useMemo, type ReactNode } from "react";

import { DataTable, DataTableRowDragHandle } from "@/components/data-table";
import { LabelPicker } from "@/components/labels/label-picker";
import { Field, FieldError } from "@/components/ui/field";
import { unavailableLabelTargetIDs, type LabelTargetRow } from "@/lib/label-target-rows";

export function LabelTargetRowsTable<T extends LabelTargetRow>({
  rows,
  labelErrors = {},
  excludeLabelIDs,
  columnsBeforeLabel = [],
  columnsAfterLabel = [],
  empty,
  onChange,
  onLabelChange,
  renderActions,
}: {
  rows: T[];
  labelErrors?: Partial<Record<number, string>>;
  excludeLabelIDs: readonly number[];
  columnsBeforeLabel?: ColumnDef<T>[];
  columnsAfterLabel?: ColumnDef<T>[];
  empty: ReactNode;
  onChange: (rows: T[]) => void;
  onLabelChange: (id: number, labelID: number | null) => void;
  renderActions?: (row: T) => ReactNode;
}) {
  const columns = useMemo<ColumnDef<T>[]>(
    () => [
      {
        id: "drag",
        header: () => null,
        enableSorting: false,
        enableHiding: false,
        cell: () => <DataTableRowDragHandle />,
        meta: { headClassName: "w-10", cellClassName: "w-10 align-top pt-3" },
      },
      ...columnsBeforeLabel,
      {
        id: "labels",
        accessorKey: "label_id",
        header: () => (
          <span className="inline-flex items-center gap-1">
            Label
            <span aria-hidden="true" className="text-destructive">
              *
            </span>
          </span>
        ),
        enableSorting: false,
        cell: ({ row }) => {
          const error = labelErrors[row.original.id];
          const unavailableLabelIDs = unavailableLabelTargetIDs(rows, excludeLabelIDs, row.original.id);

          return (
            <Field data-invalid={error ? true : undefined} className="min-w-72 gap-1">
              <LabelPicker
                value={row.original.label_id === null ? [] : [row.original.label_id]}
                selectionMode="single"
                includeBuiltins
                required
                unavailableLabelIDs={unavailableLabelIDs}
                placeholder="Select Label"
                emptyMessage="No Unused Labels Available."
                emptyPlaceholder="No Unused Labels"
                invalid={error ? true : undefined}
                onChange={(labelIDs) => onLabelChange(row.original.id, labelIDs[0] ?? null)}
              />
              {error ? <FieldError>{error}</FieldError> : null}
            </Field>
          );
        },
      },
      ...columnsAfterLabel,
      ...(renderActions
        ? [
            {
              id: "actions",
              header: () => null,
              enableSorting: false,
              enableHiding: false,
              cell: ({ row }) => renderActions(row.original),
              meta: { headClassName: "w-20", cellClassName: "w-20 align-top pt-3" },
            } satisfies ColumnDef<T>,
          ]
        : []),
    ],
    [columnsAfterLabel, columnsBeforeLabel, excludeLabelIDs, labelErrors, onLabelChange, renderActions, rows],
  );

  if (rows.length === 0) {
    return <>{empty}</>;
  }

  return (
    <DataTable
      columns={columns}
      data={rows}
      totalCount={rows.length}
      pagination={{ pageIndex: 0, pageSize: Math.max(rows.length, 1) }}
      sorting={[]}
      onPaginationChange={() => undefined}
      onSortingChange={() => undefined}
      getRowId={(row) => String(row.id)}
      clientSort
      rowReorderDisabled={rows.length <= 1}
      onRowReorder={onChange}
      empty={empty}
    />
  );
}
