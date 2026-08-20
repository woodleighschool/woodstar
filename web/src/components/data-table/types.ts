import {
  columnFilteringFeature,
  columnResizingFeature,
  columnSizingFeature,
  columnVisibilityFeature,
  createExpandedRowModel,
  metaHelper,
  rowExpandingFeature,
  rowPaginationFeature,
  rowSelectionFeature,
  rowSortingFeature,
  tableFeatures,
  type CellContext,
  type Column,
  type ColumnDef,
  type Header,
  type ReactTable,
  type Row,
  type RowData,
  type TableOptions,
  type TableState,
} from "@tanstack/react-table";

import type { FacetedFilterOption } from "@components/faceted-filter";

interface DataTableColumnMeta {
  label?: string;
  options?: Option[];
}

export const dataTableFeatures = tableFeatures({
  columnFilteringFeature,
  columnSizingFeature,
  columnResizingFeature,
  columnVisibilityFeature,
  rowExpandingFeature,
  rowPaginationFeature,
  rowSelectionFeature,
  rowSortingFeature,
  expandedRowModel: createExpandedRowModel(),
  columnMeta: metaHelper<DataTableColumnMeta>(),
});

type DataTableFeatures = typeof dataTableFeatures;
export type DataTableRowData = RowData;
export type DataTableColumnDef<TData extends RowData, TValue = unknown> = ColumnDef<
  DataTableFeatures,
  TData,
  TValue
>;
export type DataTableCellContext<TData extends RowData, TValue = unknown> = CellContext<
  DataTableFeatures,
  TData,
  TValue
>;
export type DataTableColumn<TData extends RowData, TValue = unknown> = Column<
  DataTableFeatures,
  TData,
  TValue
>;
export type DataTableHeader<TData extends RowData, TValue = unknown> = Header<
  DataTableFeatures,
  TData,
  TValue
>;
export type DataTableRow<TData extends RowData> = Row<DataTableFeatures, TData>;
export type DataTableInstance<TData extends RowData> = ReactTable<DataTableFeatures, TData>;
export type DataTableOptions<TData extends RowData> = TableOptions<DataTableFeatures, TData>;
export type DataTableState = TableState<DataTableFeatures>;

export type Option = FacetedFilterOption;
