import { getRouteApi } from "@tanstack/react-router";
import { MoreHorizontal, UserPlus, Users } from "lucide-react";
import * as React from "react";

import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import type { DataTableCellContext, DataTableColumnDef } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { EnumBadge } from "@components/enum-badge";
import { FilterChip } from "@components/filter-controls";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@components/ui/dropdown-menu";
import { useAuth } from "@features/auth/queries";
import { useGroup } from "@features/directory/groups/queries";
import { DIRECTORY_SOURCE_OPTIONS, DIRECTORY_SOURCES } from "@features/directory/source";
import { UserDeleteDialog } from "@features/directory/users/delete-dialog";
import {
  USER_ACCESS_ROLE_OPTIONS,
  USER_ACCESS_ROLES,
  userAccessRole,
} from "@features/directory/users/metadata";
import { useUsers } from "@features/directory/users/queries";
import type { User } from "@lib/api";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";
import { nonEmpty } from "@lib/utils";

const routeApi = getRouteApi("/_authenticated/directory/users/");
const USER_FILTER_KEYS = [{ id: "role", multiple: true }, { id: "source" }] as const;

interface UserTableRow {
  user: User;
  currentUserId: number | null;
  isAdmin: boolean;
  onDelete: (user: User) => void;
}

function UserNameCell({ row }: DataTableCellContext<UserTableRow>) {
  const label = nonEmpty(row.original.user.name) ?? row.original.user.email;
  return (
    <Link
      to="/directory/users/$id"
      params={{ id: String(row.original.user.id) }}
      className="font-medium"
    >
      {label}
    </Link>
  );
}

function UserEmailCell({ row }: DataTableCellContext<UserTableRow>) {
  return `${row.original.user.email}${row.original.user.id === row.original.currentUserId ? " (you)" : ""}`;
}

function UserActionsCell({ row }: DataTableCellContext<UserTableRow>) {
  return (
    <UserRowActions
      user={row.original.user}
      isSelf={row.original.user.id === row.original.currentUserId}
      onDelete={row.original.onDelete}
    />
  );
}

const userColumns: DataTableColumnDef<UserTableRow>[] = [
  {
    id: "name",
    accessorFn: (row) => row.user.name,
    header: "Name",
    cell: UserNameCell,
    enableHiding: false,
    meta: { label: "Name" },
  },
  {
    id: "email",
    accessorFn: (row) => row.user.email,
    header: "Email",
    cell: UserEmailCell,
    meta: { label: "Email" },
  },
  {
    id: "role",
    accessorFn: (row) => row.user.role,
    header: "Role",
    cell: ({ row }) => (
      <EnumBadge value={userAccessRole(row.original.user.role)} metadata={USER_ACCESS_ROLES} />
    ),
    meta: { label: "Role", options: USER_ACCESS_ROLE_OPTIONS },
    enableColumnFilter: true,
  },
  {
    id: "source",
    accessorFn: (row) => row.user.source,
    header: "Source",
    cell: ({ row }) => <EnumBadge value={row.original.user.source} metadata={DIRECTORY_SOURCES} />,
    meta: { label: "Source", options: DIRECTORY_SOURCE_OPTIONS },
    enableColumnFilter: true,
  },
  {
    id: "department",
    accessorFn: (row) => row.user.department,
    header: "Department",
    cell: ({ row }) => nonEmpty(row.original.user.department) ?? "-",
    meta: { label: "Department" },
  },
  {
    id: "actions",
    header: () => null,
    enableSorting: false,
    enableHiding: false,
    size: 44,
    minSize: 44,
    maxSize: 44,
    enableResizing: false,
    cell: UserActionsCell,
  },
];

const userViewerColumns = userColumns.filter((column) => column.id !== "actions");

export function UserListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: USER_FILTER_KEYS,
    scopeKeys: ["group_id"],
  });
  const { user: currentUser } = useAuth();
  const isAdmin = currentUser?.role === "admin";
  const [deleting, setDeleting] = React.useState<User | null>(null);
  const role = search.role;
  const source = search.source;
  const groupID = search.group_id;
  const group = useGroup(groupID ?? null);
  const query = useUsers({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    role,
    source,
    group_id: groupID,
  });
  const tableRows = React.useMemo<UserTableRow[]>(
    () =>
      query.data?.items.map((user) => ({
        user,
        currentUserId: currentUser?.id ?? null,
        isAdmin,
        onDelete: setDeleting,
      })) ?? [],
    [currentUser?.id, isAdmin, query.data?.items],
  );
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const groupLabel =
    groupID === undefined ? undefined : (group.data?.display_name ?? `Group #${groupID}`);
  const table = useDataTable({
    tableState: tableSearch,
    data: tableRows,
    columns: isAdmin ? userColumns : userViewerColumns,
    pageCount,
    rowCount: totalCount,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
    getRowId: (row) => String(row.user.id),
  });
  return (
    <PageShell>
      <PageHeader
        title="Users"
        description="Manage directory and local users."
        context={
          groupLabel ? (
            <FilterChip
              label="Group"
              value={groupLabel}
              onRemove={() => tableSearch.clearSearchKeys(["group_id"])}
            />
          ) : null
        }
        actions={
          isAdmin ? (
            <Button size="sm" render={<Link to="/directory/users/new" />} nativeButton={false}>
              <UserPlus data-icon="inline-start" />
              Create
            </Button>
          ) : null
        }
      />

      {query.error ? (
        <QueryError
          title="Failed to load users"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={6} filterCount={2} />
      ) : (
        <DataTable
          table={table}
          pending={query.isFetching}
          empty={
            <DataTableEmpty
              icon={<Users />}
              filtered={tableSearch.isFiltered}
              title="No users"
              description="Create a local user or configure directory sync."
              filteredDescription="No users matched the current filters."
            />
          }
        >
          <DataTableSearchInput
            loading={query.isFetching}
            value={tableSearch.q ?? ""}
            onValueChange={tableSearch.onQueryChange}
          />
          <DataTableFacetedFilter
            column={table.getColumn("role")}
            title="Role"
            options={USER_ACCESS_ROLE_OPTIONS}
          />
          <DataTableFacetedFilter
            column={table.getColumn("source")}
            title="Source"
            options={DIRECTORY_SOURCE_OPTIONS}
          />
        </DataTable>
      )}

      {isAdmin ? (
        <UserDeleteDialog
          open={deleting !== null}
          onOpenChange={(open) => {
            if (!open) setDeleting(null);
          }}
          user={deleting}
        />
      ) : null}
    </PageShell>
  );
}
function userEditLink(userId: number, currentUserId: number | null) {
  return userId === currentUserId
    ? ({ to: "/account" } as const)
    : ({ to: "/directory/users/$id/edit", params: { id: String(userId) } } as const);
}
function UserRowActions({
  user,
  isSelf,
  onDelete,
}: {
  user: User;
  isSelf: boolean;
  onDelete: (user: User) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button type="button" size="icon" variant="ghost" />}>
        <MoreHorizontal />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem render={<Link {...userEditLink(user.id, isSelf ? user.id : null)} />}>
            Edit
          </DropdownMenuItem>
          {!isSelf ? (
            <DropdownMenuItem variant="destructive" onClick={() => onDelete(user)}>
              Delete
            </DropdownMenuItem>
          ) : null}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
