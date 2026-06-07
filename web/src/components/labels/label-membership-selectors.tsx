import type { ColumnDef, PaginationState, SortingState } from "@tanstack/react-table";
import { ServerCog, UsersRound } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";

import { DataTable, DataTableColumnHeader, DataTableSearch } from "@/components/data-table";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useGroups, type Group } from "@/hooks/use-groups";
import { useHosts, type Host } from "@/hooks/use-hosts";
import { useUserDepartments, useUsers, type Department, type User } from "@/hooks/use-users";
import type { LabelDerivedAttribute } from "@/lib/labels";

export function HostSelector({ value, onChange }: { value: number[]; onChange: (value: number[]) => void }) {
  const controls = useSelectorControls([{ id: "display_name", desc: false }]);
  const showSelected = controls.filter === "selected";
  const hosts = useHosts({
    q: controls.q,
    page_index: controls.pagination.pageIndex,
    page_size: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    ids: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (hosts.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (hosts.data?.count ?? 0);
  const columns = useMemo<ColumnDef<Host>[]>(
    () => [
      {
        id: "display_name",
        accessorFn: (row) => row.display_name,
        header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
        cell: ({ row }) => row.original.display_name,
      },
      {
        id: "hardware.serial",
        accessorFn: (row) => row.hardware.serial,
        header: ({ column }) => <DataTableColumnHeader column={column} title="Serial" />,
        cell: ({ row }) => row.original.hardware.serial,
      },
      {
        id: "hardware.model_identifier",
        accessorFn: (row) => row.hardware.model_identifier,
        header: ({ column }) => <DataTableColumnHeader column={column} title="Model" />,
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
      pagination={controls.pagination}
      sorting={controls.sorting}
      onPaginationChange={controls.setPagination}
      onSortingChange={controls.setSorting}
      searchValue={controls.q}
      searchPlaceholder="Search Hosts"
      filter={controls.filter}
      selectedCount={value.length}
      isLoading={hosts.isLoading}
      error={hosts.error?.message}
      selectedRowIds={value.map(String)}
      onSearchChange={controls.setSearch}
      onFilterChange={controls.setSelectionFilter}
      onSelectedRowIdsChange={(ids) => onChange(ids.map(Number).filter((id) => Number.isInteger(id) && id > 0))}
      getRowId={(host) => String(host.id)}
      emptyTitle={showSelected ? "No Selected Hosts" : "No Hosts Found"}
      emptyDescription={
        showSelected ? "Selected hosts will appear here." : "Try another search term or enroll hosts first."
      }
      emptyIcon={<ServerCog className="size-5" />}
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
    case "directory_group":
      return <GroupSelector value={value} onChange={onChange} />;
    case "user":
      return <UserSelector value={value} onChange={onChange} />;
    default:
      return <DepartmentSelector value={value} onChange={onChange} />;
  }
}

function DepartmentSelector({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  const controls = useSelectorControls([{ id: "value", desc: false }]);
  const showSelected = controls.filter === "selected";
  const departments = useUserDepartments({
    q: controls.q,
    page_index: controls.pagination.pageIndex,
    page_size: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (departments.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (departments.data?.count ?? 0);
  const columns = useMemo<ColumnDef<Department>[]>(
    () => [
      {
        accessorKey: "value",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Department" />,
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
      pagination={controls.pagination}
      sorting={controls.sorting}
      onPaginationChange={controls.setPagination}
      onSortingChange={controls.setSorting}
      searchValue={controls.q}
      searchPlaceholder="Search Departments"
      filter={controls.filter}
      selectedCount={value.length}
      isLoading={departments.isLoading}
      error={departments.error?.message}
      selectedRowIds={value}
      onSearchChange={controls.setSearch}
      onFilterChange={controls.setSelectionFilter}
      onSelectedRowIdsChange={onChange}
      getRowId={(department) => department.value}
      emptyTitle={showSelected ? "No Selected Departments" : "No Departments Found"}
      emptyDescription={
        showSelected ? "Selected departments will appear here." : "Try another search term or sync users first."
      }
      emptyIcon={<UsersRound className="size-5" />}
    />
  );
}

function GroupSelector({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  const controls = useSelectorControls([{ id: "display_name", desc: false }]);
  const showSelected = controls.filter === "selected";
  const groups = useGroups({
    q: controls.q,
    page_index: controls.pagination.pageIndex,
    page_size: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (groups.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (groups.data?.count ?? 0);
  const columns = useMemo<ColumnDef<Group>[]>(
    () => [
      {
        accessorKey: "display_name",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Group" />,
        cell: ({ row }) => row.original.display_name,
      },
      {
        accessorKey: "mail_nickname",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Nickname" />,
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
      pagination={controls.pagination}
      sorting={controls.sorting}
      onPaginationChange={controls.setPagination}
      onSortingChange={controls.setSorting}
      searchValue={controls.q}
      searchPlaceholder="Search Groups"
      filter={controls.filter}
      selectedCount={value.length}
      isLoading={groups.isLoading}
      error={groups.error?.message}
      selectedRowIds={value}
      onSearchChange={controls.setSearch}
      onFilterChange={controls.setSelectionFilter}
      onSelectedRowIdsChange={onChange}
      getRowId={(group) => group.external_id}
      emptyTitle={showSelected ? "No Selected Groups" : "No Groups Found"}
      emptyDescription={
        showSelected ? "Selected groups will appear here." : "Try another search term or sync groups first."
      }
      emptyIcon={<UsersRound className="size-5" />}
    />
  );
}

function UserSelector({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  const controls = useSelectorControls([{ id: "name", desc: false }]);
  const showSelected = controls.filter === "selected";
  const users = useUsers({
    q: controls.q,
    page_index: controls.pagination.pageIndex,
    page_size: controls.pagination.pageSize,
    sort: sortParam(controls.sorting),
    values: showSelected ? value : undefined,
  });
  const rows = showSelected && value.length === 0 ? [] : (users.data?.items ?? []);
  const count = showSelected && value.length === 0 ? 0 : (users.data?.count ?? 0);
  const columns = useMemo<ColumnDef<User>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => <DataTableColumnHeader column={column} title="User" />,
        cell: ({ row }) => row.original.name,
      },
      {
        accessorKey: "department",
        header: ({ column }) => <DataTableColumnHeader column={column} title="Department" />,
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
      pagination={controls.pagination}
      sorting={controls.sorting}
      onPaginationChange={controls.setPagination}
      onSortingChange={controls.setSorting}
      searchValue={controls.q}
      searchPlaceholder="Search Users"
      filter={controls.filter}
      selectedCount={value.length}
      isLoading={users.isLoading}
      error={users.error?.message}
      selectedRowIds={value}
      onSearchChange={controls.setSearch}
      onFilterChange={controls.setSelectionFilter}
      onSelectedRowIdsChange={onChange}
      getRowId={(user) => String(user.id)}
      emptyTitle={showSelected ? "No Selected Users" : "No Users Found"}
      emptyDescription={
        showSelected ? "Selected users will appear here." : "Try another search term or sync users first."
      }
      emptyIcon={<UsersRound className="size-5" />}
    />
  );
}

type SelectionFilter = "all" | "selected";

function useSelectorControls(defaultSorting: SortingState) {
  const [q, setQ] = useState("");
  const [filter, setFilter] = useState<SelectionFilter>("all");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 });
  const [sorting, setSorting] = useState<SortingState>(defaultSorting);

  function resetPage() {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }));
  }

  const setSearch = (next: string) => {
    setQ(next);
    resetPage();
  };
  const setSelectionFilter = (next: SelectionFilter) => {
    setFilter(next);
    resetPage();
  };

  return {
    q,
    filter,
    pagination,
    sorting,
    setPagination,
    setSorting,
    setSearch,
    setSelectionFilter,
  };
}

interface SelectorTableProps<TData> {
  columns: ColumnDef<TData>[];
  data: TData[];
  totalCount: number;
  pagination: PaginationState;
  sorting: SortingState;
  onPaginationChange: (pagination: PaginationState | ((pagination: PaginationState) => PaginationState)) => void;
  onSortingChange: (sorting: SortingState | ((sorting: SortingState) => SortingState)) => void;
  searchValue: string;
  searchPlaceholder: string;
  filter: SelectionFilter;
  selectedCount: number;
  isLoading: boolean;
  error?: string;
  selectedRowIds: string[];
  onSearchChange: (value: string) => void;
  onFilterChange: (value: SelectionFilter) => void;
  onSelectedRowIdsChange: (ids: string[]) => void;
  getRowId: (row: TData) => string;
  emptyTitle: string;
  emptyDescription: string;
  emptyIcon: ReactNode;
}

function SelectorTable<TData>({
  columns,
  data,
  totalCount,
  pagination,
  sorting,
  onPaginationChange,
  onSortingChange,
  searchValue,
  searchPlaceholder,
  filter,
  selectedCount,
  isLoading,
  error,
  selectedRowIds,
  onSearchChange,
  onFilterChange,
  onSelectedRowIdsChange,
  getRowId,
  emptyTitle,
  emptyDescription,
  emptyIcon,
}: SelectorTableProps<TData>) {
  if (error) {
    return <p className="text-destructive text-sm">{error}</p>;
  }

  return (
    <DataTable
      columns={columns}
      data={data}
      totalCount={totalCount}
      pagination={pagination}
      sorting={sorting}
      onPaginationChange={onPaginationChange}
      onSortingChange={onSortingChange}
      isLoading={isLoading}
      enableRowSelection
      selectedRowIds={selectedRowIds}
      onSelectedRowIdsChange={onSelectedRowIdsChange}
      getRowId={getRowId}
      perPageOptions={[10, 25, 50]}
      toolbar={
        <div className="flex flex-wrap items-center gap-2">
          <DataTableSearch
            value={searchValue}
            onChange={onSearchChange}
            placeholder={searchPlaceholder}
            className="min-w-64"
          />
          <ToggleGroup
            type="single"
            value={filter}
            onValueChange={(next) => {
              if (next === "all" || next === "selected") onFilterChange(next);
            }}
            variant="outline"
            size="sm"
          >
            <ToggleGroupItem value="all">Show All</ToggleGroupItem>
            <ToggleGroupItem value="selected">Selected {selectedCount}</ToggleGroupItem>
          </ToggleGroup>
        </div>
      }
      empty={
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">{emptyIcon}</EmptyMedia>
            <EmptyTitle>{emptyTitle}</EmptyTitle>
            <EmptyDescription>{emptyDescription}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      }
    />
  );
}

function sortParam(sorting: SortingState) {
  if (sorting.length === 0) return undefined;
  const first = sorting[0];
  return `${first.id}.${first.desc ? "desc" : "asc"}`;
}
