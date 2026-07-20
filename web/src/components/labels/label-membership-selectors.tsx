import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  type PaginationState,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table";
import { useMemo, useState } from "react";

import { EmptyPanel } from "@/components/empty-panel";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { encodeSort } from "@/hooks/use-data-table-search";
import { useGroups } from "@/hooks/use-groups";
import { useHosts } from "@/hooks/use-hosts";
import { useUserDepartments, useUsers } from "@/hooks/use-users";
import type { Department, Group, Host, User } from "@/lib/api";
import type { LabelDerivedAttribute } from "@/lib/labels";
import { assertNever } from "@/lib/utils";

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
  const columns = useMemo<ColumnDef<Host>[]>(
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
      searchPlaceholder="Search hosts"
      selectedCount={value.length}
      isLoading={hosts.isLoading}
      error={hosts.error?.message}
      selectedRowIds={value.map(String)}
      onSelectedRowIdsChange={(ids) =>
        onChange(ids.map(Number).filter((id) => Number.isInteger(id) && id > 0))
      }
      getRowId={(host) => String(host.id)}
      emptyTitle={showSelected ? "No selected hosts" : "No hosts found"}
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
  const columns = useMemo<ColumnDef<Department>[]>(
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
      searchPlaceholder="Search departments"
      selectedCount={value.length}
      isLoading={departments.isLoading}
      error={departments.error?.message}
      selectedRowIds={value}
      onSelectedRowIdsChange={onChange}
      getRowId={(department) => department.value}
      emptyTitle={showSelected ? "No selected departments" : "No departments found"}
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
  const columns = useMemo<ColumnDef<Group>[]>(
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
      searchPlaceholder="Search groups"
      selectedCount={value.length}
      isLoading={groups.isLoading}
      error={groups.error?.message}
      selectedRowIds={value}
      onSelectedRowIdsChange={onChange}
      getRowId={(group) => group.external_id}
      emptyTitle={showSelected ? "No selected groups" : "No groups found"}
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
  const columns = useMemo<ColumnDef<User>[]>(
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
      searchPlaceholder="Search users"
      selectedCount={value.length}
      isLoading={users.isLoading}
      error={users.error?.message}
      selectedRowIds={value}
      onSelectedRowIdsChange={onChange}
      getRowId={(user) => String(user.id)}
      emptyTitle={showSelected ? "No selected users" : "No users found"}
    />
  );
}

type SelectionFilter = "all" | "selected";

interface SelectorControls {
  q: string;
  filter: SelectionFilter;
  pagination: PaginationState;
  sorting: SortingState;
  setPagination: React.Dispatch<React.SetStateAction<PaginationState>>;
  setSorting: React.Dispatch<React.SetStateAction<SortingState>>;
  setSearch: (next: string) => void;
  setSelectionFilter: (next: SelectionFilter) => void;
}

function useSelectorControls(defaultSorting: SortingState): SelectorControls {
  const [q, setQ] = useState("");
  const [filter, setFilter] = useState<SelectionFilter>("all");
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 10 });
  const [sorting, setSorting] = useState<SortingState>(defaultSorting);

  function resetPage() {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }));
  }

  return {
    q,
    filter,
    pagination,
    sorting,
    setPagination,
    setSorting,
    setSearch: (next: string) => {
      setQ(next);
      resetPage();
    },
    setSelectionFilter: (next: SelectionFilter) => {
      setFilter(next);
      resetPage();
    },
  };
}

interface SelectorTableProps<TData> {
  columns: ColumnDef<TData>[];
  data: TData[];
  totalCount: number;
  controls: SelectorControls;
  searchPlaceholder: string;
  selectedCount: number;
  isLoading: boolean;
  error?: string;
  selectedRowIds: string[];
  onSelectedRowIdsChange: (ids: string[]) => void;
  getRowId: (row: TData) => string;
  emptyTitle: string;
}

// Server-paginated multi-select picker with local (non-URL) state. Selection is
// kept externally as id strings so it survives paging; row checkboxes read/write
// that set directly rather than TanStack's page-scoped rowSelection.
function SelectorTable<TData>({
  columns,
  data,
  totalCount,
  controls,
  searchPlaceholder,
  selectedCount,
  isLoading,
  error,
  selectedRowIds,
  onSelectedRowIdsChange,
  getRowId,
  emptyTitle,
}: SelectorTableProps<TData>) {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getRowId: (row) => getRowId(row),
    manualPagination: true,
    manualSorting: true,
    enableMultiSort: false,
    pageCount: Math.max(1, Math.ceil(totalCount / controls.pagination.pageSize)),
    state: { pagination: controls.pagination, sorting: controls.sorting },
    onPaginationChange: controls.setPagination,
    onSortingChange: (updater) => {
      controls.setSorting((prev) =>
        singleSort(typeof updater === "function" ? updater(prev) : updater),
      );
    },
  });

  const selectedSet = new Set(selectedRowIds);
  const pageRows = table.getRowModel().rows;
  const pageIds = pageRows.map((row) => getRowId(row.original));
  const allPageSelected = pageIds.length > 0 && pageIds.every((id) => selectedSet.has(id));
  const pageCount = Math.max(1, Math.ceil(totalCount / controls.pagination.pageSize));

  function togglePage(checked: boolean) {
    if (checked) onSelectedRowIdsChange([...new Set([...selectedRowIds, ...pageIds])]);
    else onSelectedRowIdsChange(selectedRowIds.filter((id) => !pageIds.includes(id)));
  }
  function toggleRow(id: string, checked: boolean) {
    if (checked) onSelectedRowIdsChange([...selectedRowIds, id]);
    else onSelectedRowIdsChange(selectedRowIds.filter((existing) => existing !== id));
  }

  if (error) {
    return <QueryError title="Failed to load options" error={{ message: error }} />;
  }

  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={controls.q}
          onChange={(event) => controls.setSearch(event.target.value)}
          placeholder={searchPlaceholder}
          className="h-8 min-w-64 flex-1"
        />
        <ToggleGroup
          value={[controls.filter]}
          onValueChange={(next) => {
            const value = next[0];
            if (value === "all" || value === "selected") controls.setSelectionFilter(value);
          }}
          variant="outline"
          size="sm"
        >
          <ToggleGroupItem value="all">Show all</ToggleGroupItem>
          <ToggleGroupItem value="selected">Selected {selectedCount}</ToggleGroupItem>
        </ToggleGroup>
      </div>

      <div className="overflow-x-auto rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                <TableHead className="w-10">
                  <Checkbox checked={allPageSelected} onCheckedChange={togglePage} />
                </TableHead>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {!isLoading && pageRows.length ? (
              pageRows.map((row) => {
                const id = getRowId(row.original);
                return (
                  <TableRow key={row.id} data-state={selectedSet.has(id) ? "selected" : undefined}>
                    <TableCell className="w-10">
                      <Checkbox
                        checked={selectedSet.has(id)}
                        onCheckedChange={(value) => toggleRow(id, value)}
                      />
                    </TableCell>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                );
              })
            ) : (
              <TableRow className="hover:bg-transparent">
                <TableCell colSpan={columns.length + 1} className="p-0">
                  {isLoading ? (
                    <div className="h-24 text-center leading-24 text-muted-foreground">
                      Loading...
                    </div>
                  ) : (
                    <EmptyPanel>{emptyTitle}</EmptyPanel>
                  )}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <div className="flex items-center justify-between gap-2">
        <span className="text-sm text-muted-foreground">{totalCount} total</span>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={controls.pagination.pageIndex <= 0}
            onClick={() =>
              controls.setPagination((prev) => ({ ...prev, pageIndex: prev.pageIndex - 1 }))
            }
          >
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {controls.pagination.pageIndex + 1} of {pageCount}
          </span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={controls.pagination.pageIndex >= pageCount - 1}
            onClick={() =>
              controls.setPagination((prev) => ({ ...prev, pageIndex: prev.pageIndex + 1 }))
            }
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}

function sortParam(sorting: SortingState) {
  if (sorting.length === 0) return undefined;
  return encodeSort(sorting[0].id, sorting[0].desc);
}

function singleSort(sorting: SortingState): SortingState {
  return sorting.length > 0 ? [sorting[0]] : [];
}
