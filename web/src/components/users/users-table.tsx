import { Link } from "@tanstack/react-router";
import type { ColumnDef, OnChangeFn, PaginationState, SortingState } from "@tanstack/react-table";
import { MoreHorizontal, Users } from "lucide-react";
import type { ReactNode } from "react";

import {
  DataTable,
  DataTableColumnHeader,
  DataTableEmptyState,
  DataTableFacetedFilter,
  DataTableSearch,
} from "@/components/data-table";
import { EnumBadge } from "@/components/enum-badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { User, useUsers } from "@/hooks/use-users";
import { DIRECTORY_SOURCE_OPTIONS, DIRECTORY_SOURCES } from "@/lib/directory";
import { USER_ACCESS_ROLE_OPTIONS, USER_ACCESS_ROLES, userAccessRole } from "@/lib/users";
import { nonEmpty } from "@/lib/utils";

interface UsersTableProps {
  data: User[];
  totalCount: number;
  query: ReturnType<typeof useUsers>;
  currentUserId: number | null;
  pagination: PaginationState;
  sorting: SortingState;
  onPaginationChange: OnChangeFn<PaginationState>;
  onSortingChange: OnChangeFn<SortingState>;
  toolbar: ReactNode;
  hasFilters: boolean;
  onDelete: (user: User) => void;
}

export function UsersTable({
  data,
  totalCount,
  query,
  currentUserId,
  pagination,
  sorting,
  onPaginationChange,
  onSortingChange,
  toolbar,
  hasFilters,
  onDelete,
}: UsersTableProps) {
  const columns: ColumnDef<User>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => nonEmpty(row.original.name) ?? row.original.email,
    },
    {
      id: "email",
      accessorKey: "email",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Email" />,
      cell: ({ row }) => {
        const isSelf = row.original.id === currentUserId;
        return `${row.original.email}${isSelf ? " (you)" : ""}`;
      },
    },
    {
      id: "role",
      accessorKey: "role",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Role" />,
      cell: ({ row }) => <EnumBadge value={userAccessRole(row.original.role)} metadata={USER_ACCESS_ROLES} />,
    },
    {
      id: "source",
      accessorKey: "source",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Source" />,
      cell: ({ row }) => <EnumBadge value={row.original.source} metadata={DIRECTORY_SOURCES} />,
    },
    {
      id: "department",
      accessorKey: "department",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Department" />,
      cell: ({ row }) => nonEmpty(row.original.department) ?? "-",
    },
    {
      id: "actions",
      header: () => null,
      enableSorting: false,
      cell: ({ row }) => (
        <UserRowActions user={row.original} isSelf={row.original.id === currentUserId} onDelete={onDelete} />
      ),
      meta: { headClassName: "w-12" },
    },
  ];

  if (query.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to Load Users</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
        <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
          Retry
        </Button>
      </Alert>
    );
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
      isLoading={query.isLoading}
      toolbar={toolbar}
      rowHref={(row) =>
        row.id === currentUserId
          ? { to: "/account" }
          : { to: "/directory/users/$userId/edit", params: { userId: String(row.id) } }
      }
      empty={
        <DataTableEmptyState
          icon={<Users />}
          title={hasFilters ? "No Matches" : "No Users"}
          description={hasFilters ? "No users matched the current filters." : "Users appear after setup or sync."}
        />
      }
    />
  );
}

export function UsersToolbar({
  draft,
  onDraftChange,
  role,
  source,
  onFilterChange,
}: {
  draft: string;
  onDraftChange: (next: string) => void;
  role?: string;
  source?: string;
  onFilterChange: (key: string, value: string | undefined) => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <DataTableSearch value={draft} onChange={onDraftChange} placeholder="Search" />
      <DataTableFacetedFilter
        title="Role"
        options={USER_ACCESS_ROLE_OPTIONS}
        selected={role ? [role] : []}
        onChange={(next) => onFilterChange("role", next[0])}
        singleSelect
      />
      <DataTableFacetedFilter
        title="Source"
        options={DIRECTORY_SOURCE_OPTIONS}
        selected={source ? [source] : []}
        onChange={(next) => onFilterChange("source", next[0])}
        singleSelect
      />
    </div>
  );
}

function UserRowActions({ user, isSelf, onDelete }: { user: User; isSelf: boolean; onDelete: (user: User) => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" size="icon" variant="ghost">
          <MoreHorizontal />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          {isSelf ? (
            <DropdownMenuItem asChild>
              <Link to="/account">Edit</Link>
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem asChild>
              <Link to="/directory/users/$userId/edit" params={{ userId: String(user.id) }}>
                Edit
              </Link>
            </DropdownMenuItem>
          )}
          {!isSelf ? (
            <DropdownMenuItem variant="destructive" onSelect={() => onDelete(user)}>
              Delete
            </DropdownMenuItem>
          ) : null}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
