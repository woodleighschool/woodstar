import { Link } from "@tanstack/react-router";
import type { ColumnDef, PaginationState, SortingState } from "@tanstack/react-table";
import { MoreHorizontal, UserPlus, Users } from "lucide-react";
import { useState } from "react";

import { DataTable, DataTableColumnHeader, DataTableEmptyState } from "@/components/data-table";
import { EnumBadge } from "@/components/enum-badge";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { USER_ACCESS_ROLES, userAccessRole } from "@/components/users/user-role";
import { useAuth } from "@/hooks/use-auth";
import { useUsers, type User } from "@/hooks/use-users";
import { formatRelative } from "@/lib/utils";

const USERS_TABLE_PAGINATION: PaginationState = { pageIndex: 0, pageSize: 100 };
const USERS_TABLE_SORTING: SortingState = [];

export function UsersPage() {
  const query = useUsers();
  const { user: currentUser } = useAuth();

  const [createOpen, setCreateOpen] = useState(false);
  const [deleting, setDeleting] = useState<User | null>(null);

  return (
    <PageShell>
      <PageHeader
        title="Users"
        description="Manage users and Woodstar access."
        actions={
          <Button size="sm" className="gap-2" onClick={() => setCreateOpen(true)}>
            <UserPlus data-icon="inline-start" /> Create
          </Button>
        }
      />

      <div>
        <UsersTable query={query} currentUserId={currentUser?.id ?? null} onDelete={setDeleting} />
      </div>

      <UserFormDialog mode="create" open={createOpen} onOpenChange={setCreateOpen} />

      <UserDeleteDialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
        user={deleting}
      />
    </PageShell>
  );
}

interface UsersTableProps {
  query: ReturnType<typeof useUsers>;
  currentUserId: number | null;
  onDelete: (user: User) => void;
}

function UsersTable({ query, currentUserId, onDelete }: UsersTableProps) {
  const data = query.data ?? [];
  const columns: ColumnDef<User>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => row.original.name || row.original.email,
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
      id: "created_at",
      accessorKey: "created_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
      cell: ({ row }) => formatRelative(row.original.created_at),
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
      totalCount={data.length}
      pagination={USERS_TABLE_PAGINATION}
      sorting={USERS_TABLE_SORTING}
      onPaginationChange={() => undefined}
      onSortingChange={() => undefined}
      isLoading={query.isLoading}
      clientSort
      rowHref={(row) =>
        row.id === currentUserId
          ? { to: "/account" }
          : { to: "/users/$userId/edit", params: { userId: String(row.id) } }
      }
      empty={<DataTableEmptyState icon={<Users />} title="No Users" description="Create a local account." />}
    />
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
              <Link to="/users/$userId/edit" params={{ userId: String(user.id) }}>
                Edit
              </Link>
            </DropdownMenuItem>
          )}
          {!isSelf ? (
            <DropdownMenuItem variant="destructive" onSelect={() => onDelete(user)}>
              {user.synced ? "Deactivate" : "Delete"}
            </DropdownMenuItem>
          ) : null}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
