import {
  type ColumnFiltersState,
  type PaginationState,
  type RowSelectionState,
  type SortingState,
  type Updater,
  useTable,
} from "@tanstack/react-table";
import { useMemo, useState } from "react";

import { DataTable } from "@components/data-table/data-table";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DATA_TABLE_DEFAULT_COLUMN } from "@components/data-table/data-table-sizing";
import { selectColumn } from "@components/data-table/select-column";
import {
  dataTableFeatures,
  type DataTableColumnDef,
  type DataTableRowData,
} from "@components/data-table/types";
import { encodeSort } from "@components/data-table/use-data-table-search";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { useGroups } from "@features/directory/groups/queries";
import { useUserDepartments, useUsers } from "@features/directory/users/queries";
import { useHosts } from "@features/hosts/queries";
import type { LabelDerivedAttribute } from "@features/labels/model";
import type { Department, Group, Host, User } from "@lib/api";
import { PAGE_SIZE_OPTIONS } from "@lib/pagination";
import { assertNever } from "@lib/utils";

export function HostSelector({
  value,
  onChange,
}: {
  value: number[];
  onChange: (value: number[]) => void;
}) {
  const controls = useSelectorControls([{ id: "display_name", desc: false }]);
  const showSelected = controls.filter === "selected";
  const hosts = useHosts({
    q: controls.q,
    page: controls.pagination.pageIndex + 1,
    per_page: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    ids: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (hosts.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (hosts.data?.count ?? 0);
  const columns = useMemo<DataTableColumnDef<Host>[]>(
    () => [
      {
        id: "display_name",
        accessorFn: (row) => row.display_name,
        header: "Host",
        cell: ({ row }) => row.original.display_name,
      },
      {
        id: "hardware.serial",
        accessorFn: (row) => row.hardware.serial,
        header: "Serial",
        cell: ({ row }) => row.original.hardware.serial,
      },
      {
        id: "hardware.model_identifier",
        accessorFn: (row) => row.hardware.model_identifier,
        header: "Model",
        cell: ({ row }) => row.original.hardware.model_identifier || "Unknown",
      },
    ],
    [],
  );

  return (
    <SelectorTable
      columns={columns}
      data={rows}
      totalCount={count}
      controls={controls}
      searchPlaceholder="Search Hosts"
      selectedCount={value.length}
      isLoading={hosts.isLoading}
      isPlaceholderData={hosts.isPlaceholderData}
      error={hosts.error?.message}
      selectedRowIds={value.map(String)}
      onSelectedRowIdsChange={(ids) =>
        onChange(ids.map(Number).filter((id) => Number.isInteger(id) && id > 0))
      }
      getRowId={(host) => String(host.id)}
      emptyTitle={showSelected ? "No Selected Hosts" : "No Hosts Found"}
    />
  );
}

export function DerivedSelector({
  attribute,
  value,
  onChange,
}: {
  attribute: LabelDerivedAttribute;
  value: string[];
  onChange: (value: string[]) => void;
}) {
  switch (attribute) {
    case "user_department":
      return <DepartmentSelector value={value} onChange={onChange} />;
    case "directory_group":
      return <GroupSelector value={value} onChange={onChange} />;
    case "user":
      return <UserSelector value={value} onChange={onChange} />;
  }
  return assertNever(attribute);
}

function DepartmentSelector({
  value,
  onChange,
}: {
  value: string[];
  onChange: (value: string[]) => void;
}) {
  const controls = useSelectorControls([{ id: "value", desc: false }]);
  const showSelected = controls.filter === "selected";
  const departments = useUserDepartments({
    q: controls.q,
    page: controls.pagination.pageIndex + 1,
    per_page: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (departments.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (departments.data?.count ?? 0);
  const columns = useMemo<DataTableColumnDef<Department>[]>(
    () => [
      {
        accessorKey: "value",
        header: "Department",
        cell: ({ row }) => row.original.value,
      },
    ],
    [],
  );

  return (
    <SelectorTable
      columns={columns}
      data={rows}
      totalCount={count}
      controls={controls}
      searchPlaceholder="Search Departments"
      selectedCount={value.length}
      isLoading={departments.isLoading}
      isPlaceholderData={departments.isPlaceholderData}
      error={departments.error?.message}
      selectedRowIds={value}
      onSelectedRowIdsChange={onChange}
      getRowId={(department) => department.value}
      emptyTitle={showSelected ? "No Selected Departments" : "No Departments Found"}
    />
  );
}

function GroupSelector({
  value,
  onChange,
}: {
  value: string[];
  onChange: (value: string[]) => void;
}) {
  const controls = useSelectorControls([{ id: "display_name", desc: false }]);
  const showSelected = controls.filter === "selected";
  const groups = useGroups({
    q: controls.q,
    page: controls.pagination.pageIndex + 1,
    per_page: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (groups.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (groups.data?.count ?? 0);
  const columns = useMemo<DataTableColumnDef<Group>[]>(
    () => [
      {
        accessorKey: "display_name",
        header: "Group",
        cell: ({ row }) => row.original.display_name,
      },
      {
        accessorKey: "mail_nickname",
        header: "Nickname",
        cell: ({ row }) => row.original.mail_nickname ?? "None",
      },
    ],
    [],
  );

  return (
    <SelectorTable
      columns={columns}
      data={rows}
      totalCount={count}
      controls={controls}
      searchPlaceholder="Search Groups"
      selectedCount={value.length}
      isLoading={groups.isLoading}
      isPlaceholderData={groups.isPlaceholderData}
      error={groups.error?.message}
      selectedRowIds={value}
      onSelectedRowIdsChange={onChange}
      getRowId={(group) => group.external_id}
      emptyTitle={showSelected ? "No Selected Groups" : "No Groups Found"}
    />
  );
}

function UserSelector({
  value,
  onChange,
}: {
  value: string[];
  onChange: (value: string[]) => void;
}) {
  const controls = useSelectorControls([{ id: "name", desc: false }]);
  const showSelected = controls.filter === "selected";
  const users = useUsers({
    q: controls.q,
    page: controls.pagination.pageIndex + 1,
    per_page: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (users.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (users.data?.count ?? 0);
  const columns = useMemo<DataTableColumnDef<User>[]>(
    () => [
      {
        accessorKey: "name",
        header: "User",
        cell: ({ row }) => row.original.name,
      },
      {
        accessorKey: "department",
        header: "Department",
        cell: ({ row }) => row.original.department ?? "None",
      },
    ],
    [],
  );

  return (
    <SelectorTable
      columns={columns}
      data={rows}
      totalCount={count}
      controls={controls}
      searchPlaceholder="Search Users"
      selectedCount={value.length}
      isLoading={users.isLoading}
      isPlaceholderData={users.isPlaceholderData}
      error={users.error?.message}
      selectedRowIds={value}
      onSelectedRowIdsChange={onChange}
      getRowId={(user) => String(user.id)}
      emptyTitle={showSelected ? "No Selected Users" : "No Users Found"}
    />
  );
}

type SelectionFilter = "all" | "selected";

const SELECTOR_PAGE_SIZE = 10;
const SELECTOR_PAGE_SIZE_OPTIONS = [SELECTOR_PAGE_SIZE, ...PAGE_SIZE_OPTIONS] as const;

interface SelectorControls {
  q: string;
  filter: SelectionFilter;
  pagination: PaginationState;
  sorting: SortingState;
  columnFilters: ColumnFiltersState;
  setPagination: React.Dispatch<React.SetStateAction<PaginationState>>;
  setSorting: React.Dispatch<React.SetStateAction<SortingState>>;
  setColumnFilters: (updater: Updater<ColumnFiltersState>) => void;
  setSearch: (next: string | undefined) => void;
}

function useSelectorControls(defaultSorting: SortingState): SelectorControls {
  const [q, setQ] = useState("");
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: SELECTOR_PAGE_SIZE,
  });
  const [sorting, setSorting] = useState<SortingState>(defaultSorting);
  const selectionFilter = columnFilters.find((item) => item.id === "select")?.value;
  const filter: SelectionFilter =
    Array.isArray(selectionFilter) && selectionFilter.includes("selected") ? "selected" : "all";

  function resetPage() {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }));
  }

  return {
    q,
    filter,
    pagination,
    sorting,
    columnFilters,
    setPagination,
    setSorting,
    setSearch: (next: string | undefined) => {
      setQ(next ?? "");
      resetPage();
    },
    setColumnFilters: (updater) => {
      setColumnFilters((previous) => (typeof updater === "function" ? updater(previous) : updater));
      resetPage();
    },
  };
}

interface SelectorTableProps<TData extends DataTableRowData> {
  columns: DataTableColumnDef<TData>[];
  data: TData[];
  totalCount: number;
  controls: SelectorControls;
  searchPlaceholder: string;
  selectedCount: number;
  isLoading: boolean;
  isPlaceholderData: boolean;
  error?: string;
  selectedRowIds: string[];
  onSelectedRowIdsChange: (ids: string[]) => void;
  getRowId: (row: TData) => string;
  emptyTitle: string;
}

// Server-paginated multi-select picker with local (non-URL) state. Selection
// remains externally owned so it survives paging and selected-only filtering.
function SelectorTable<TData extends DataTableRowData>({
  columns,
  data,
  totalCount,
  controls,
  searchPlaceholder,
  selectedCount,
  isLoading,
  isPlaceholderData,
  error,
  selectedRowIds,
  onSelectedRowIdsChange,
  getRowId,
  emptyTitle,
}: SelectorTableProps<TData>) {
  const rowSelection = Object.fromEntries(
    selectedRowIds.map((id) => [id, true]),
  ) satisfies RowSelectionState;
  const tableData = useMemo(() => (isLoading ? [] : data), [data, isLoading]);
  const selectorColumns = useMemo(() => [selectColumn<TData>(), ...columns], [columns]);
  const table = useTable({
    features: dataTableFeatures,
    data: tableData,
    columns: selectorColumns,
    getRowId: (row) => getRowId(row),
    manualPagination: true,
    manualSorting: true,
    enableMultiSort: false,
    defaultColumn: DATA_TABLE_DEFAULT_COLUMN,
    enableColumnResizing: true,
    columnResizeMode: "onChange",
    enableRowSelection: true,
    pageCount: Math.max(1, Math.ceil(totalCount / controls.pagination.pageSize)),
    rowCount: totalCount,
    state: {
      pagination: controls.pagination,
      rowSelection,
      sorting: controls.sorting,
      columnFilters: controls.columnFilters,
    },
    onPaginationChange: controls.setPagination,
    onRowSelectionChange: (updater) => {
      const next = typeof updater === "function" ? updater(rowSelection) : updater;
      onSelectedRowIdsChange(Object.keys(next).filter((id) => next[id]));
    },
    onSortingChange: (updater) => {
      controls.setSorting((prev) =>
        singleSort(typeof updater === "function" ? updater(prev) : updater),
      );
    },
    onColumnFiltersChange: controls.setColumnFilters,
    manualFiltering: true,
  });

  if (error) {
    return <QueryError title="Failed to Load Options" error={{ message: error }} />;
  }

  return (
    <DataTable
      table={table}
      pending={isPlaceholderData}
      pageSizeOptions={SELECTOR_PAGE_SIZE_OPTIONS}
      empty={
        isLoading ? (
          <div className="h-24 text-center leading-24 text-muted-foreground">Loading...</div>
        ) : (
          <PanelEmptyState>{emptyTitle}</PanelEmptyState>
        )
      }
    >
      <DataTableSearchInput
        value={controls.q}
        onValueChange={controls.setSearch}
        loading={isLoading || isPlaceholderData}
        placeholder={searchPlaceholder}
      />
      <DataTableFacetedFilter
        column={table.getColumn("select")}
        title="Selection"
        options={[{ label: "Selected", value: "selected", count: selectedCount }]}
      />
    </DataTable>
  );
}

function sortParam(sorting: SortingState) {
  if (sorting.length === 0) return undefined;
  return encodeSort(sorting[0].id, sorting[0].desc);
}

function singleSort(sorting: SortingState): SortingState {
  return sorting.length > 0 ? [sorting[0]] : [];
}
