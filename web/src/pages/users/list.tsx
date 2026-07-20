import { Link } from "@tanstack/react-router";
import type { CellContext, ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal, UserPlus, Users } from "lucide-react";
import { parseAsInteger, useQueryStates } from "nuqs";
import * as React from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableEmpty } from "@/components/data-table/data-table-empty";
import { DataTableFacetedFilter } from "@/components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@/components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@/components/data-table/data-table-skeleton";
import { EnumBadge } from "@/components/enum-badge";
import { FilterChip } from "@/components/filter-controls";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { UserDeleteDialog } from "@/components/users/user-delete-dialog";
import { UserFormDialog } from "@/components/users/user-form-dialog";
import { useAuth } from "@/hooks/use-auth";
import { useDataTable } from "@/hooks/use-data-table";
import { useDataTableSearch } from "@/hooks/use-data-table-search";
import { useGroup } from "@/hooks/use-groups";
import { useUsers } from "@/hooks/use-users";
import type { User } from "@/lib/api";
import {
  DIRECTORY_SOURCE_OPTIONS,
  DIRECTORY_SOURCES,
  DIRECTORY_SOURCE_VALUES,
} from "@/lib/directory";
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import {
  USER_ACCESS_ROLE_OPTIONS,
  USER_ACCESS_ROLES,
  USER_ACCESS_ROLE_VALUES,
  userAccessRole,
} from "@/lib/users";
import { isOneOf, nonEmpty } from "@/lib/utils";
const USER_FILTER_KEYS = [{ id: "role" }, { id: "source" }] as const;

interface UserTableRow {
  user: User;
  currentUserId: number | null;
  isAdmin: boolean;
  onDelete: (user: User) => void;
}

function UserNameCell({ row }: CellContext<UserTableRow, unknown>) {
  const label = nonEmpty(row.original.user.name) ?? row.original.user.email;
  if (row.original.isAdmin || row.original.user.id === row.original.currentUserId) {
    return (
      <Link
        {...userEditLink(row.original.user.id, row.original.currentUserId)}
        className="font-medium hover:underline"
      >
        {label}
      </Link>
    );
  }
  return <span className="font-medium">{label}</span>;
}

function UserEmailCell({ row }: CellContext<UserTableRow, unknown>) {
  return `${row.original.user.email}${row.original.user.id === row.original.currentUserId ? " (you)" : ""}`;
}

function UserActionsCell({ row }: CellContext<UserTableRow, unknown>) {
  return (
    <UserRowActions
      user={row.original.user}
      isSelf={row.original.user.id === row.original.currentUserId}
      onDelete={row.original.onDelete}
    />
  );
}

const userColumns: ColumnDef<UserTableRow>[] = [
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
    size: 48,
    cell: UserActionsCell,
  },
];

const userViewerColumns = userColumns.filter((column) => column.id !== "actions");

export function UserListPage() {
  const tableSearch = useDataTableSearch(USER_FILTER_KEYS);
  const { user: currentUser } = useAuth();
  const isAdmin = currentUser?.role === "admin";
  const [createOpen, setCreateOpen] = React.useState(false);
  const [deleting, setDeleting] = React.useState<User | null>(null);
  const [deepLink, setDeepLink] = useQueryStates({ group_id: parseAsInteger });
  const rawRole = tableSearch.filters.role?.[0];
  const role = isOneOf(rawRole, USER_ACCESS_ROLE_VALUES) ? rawRole : undefined;
  const rawSource = tableSearch.filters.source?.[0];
  const source = isOneOf(rawSource, DIRECTORY_SOURCE_VALUES) ? rawSource : undefined;
  const groupID = deepLink.group_id ?? undefined;
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
  const users = query.data?.items ?? [];
  const tableRows: UserTableRow[] = users.map((user) => ({
    user,
    currentUserId: currentUser?.id ?? null,
    isAdmin,
    onDelete: setDeleting,
  }));
  const totalCount = query.data?.count ?? 0;
  const pageCount = query.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const hasFilters = !!tableSearch.q || !!role || !!source || groupID !== undefined;
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
              onRemove={() => void setDeepLink({ group_id: null })}
            />
          ) : null
        }
        actions={
          isAdmin ? (
            <Button size="sm" onClick={() => setCreateOpen(true)}>
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
          empty={
            <DataTableEmpty
              icon={<Users />}
              filtered={hasFilters}
              title="No users"
              description="Create a local user or configure directory sync."
              filteredDescription="No users matched the current filters."
            />
          }
        >
          <div className="flex items-start justify-between gap-2 p-1">
            <div className="flex flex-1 flex-wrap items-center gap-2">
              <DataTableSearchInput className="h-8 w-40 lg:w-56" />
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
            </div>
          </div>
        </DataTable>
      )}

      {isAdmin ? (
        <>
          <UserFormDialog open={createOpen} onOpenChange={setCreateOpen} />
          <UserDeleteDialog
            open={deleting !== null}
            onOpenChange={(open) => {
              if (!open) setDeleting(null);
            }}
            user={deleting}
          />
        </>
      ) : null}
    </PageShell>
  );
}
function userEditLink(userId: number, currentUserId: number | null) {
  return userId === currentUserId
    ? ({ to: "/account" } as const)
    : ({ to: "/directory/users/$userId/edit", params: { userId: String(userId) } } as const);
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
