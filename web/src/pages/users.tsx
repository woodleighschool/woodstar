import { MoreHorizontal, UserPlus, Users } from "lucide-react";
import { useState } from "react";

import { ErrorState } from "@/components/feedback/error-state";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/data-table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Spinner } from "@/components/ui/spinner";
import { UserDeleteDialog } from "@/components/users/user-delete-dialog";
import { UserFormDialog } from "@/components/users/user-form-dialog";
import { useAuth } from "@/hooks/use-auth";
import { useUsers, type User } from "@/hooks/use-users";
import { formatRelative } from "@/lib/utils";

export function UsersPage() {
  const query = useUsers();
  const { user: currentUser } = useAuth();

  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const [deleting, setDeleting] = useState<User | null>(null);

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Users"
        description="Local Woodstar accounts. Admins manage users and secrets; viewers are read-only."
        actions={
          <Button size="sm" className="gap-2" onClick={() => setCreateOpen(true)}>
            <UserPlus className="size-4" /> Add user
          </Button>
        }
      />

      <div className="p-6">
        <UsersTable query={query} currentUserId={currentUser?.id ?? null} onEdit={setEditing} onDelete={setDeleting} />
      </div>

      <UserFormDialog mode="create" open={createOpen} onOpenChange={setCreateOpen} />

      {editing ? (
        <UserFormDialog
          mode="edit"
          open={editing !== null}
          onOpenChange={(open) => {
            if (!open) setEditing(null);
          }}
          user={editing}
          canChangeRole={editing.id !== currentUser?.id}
        />
      ) : null}

      <UserDeleteDialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
        user={deleting}
      />
    </div>
  );
}

interface UsersTableProps {
  query: ReturnType<typeof useUsers>;
  currentUserId: string | null;
  onEdit: (user: User) => void;
  onDelete: (user: User) => void;
}

function UsersTable({ query, currentUserId, onEdit, onDelete }: UsersTableProps) {
  if (query.error) {
    return <ErrorState message={query.error.message} onRetry={() => query.refetch()} />;
  }

  if (query.isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Spinner /> Loading…
      </div>
    );
  }

  const data = query.data ?? [];
  if (data.length === 0) {
    return (
      <EmptyState
        icon={Users}
        title="No users yet"
        description="Add a user to give other admins or viewers access to this Woodstar deployment."
      />
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Email</TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Created</TableHead>
            <TableHead className="w-12" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row) => {
            const isSelf = row.id === currentUserId;
            return (
              <TableRow key={row.id}>
                <TableCell className="font-medium">
                  {row.email}
                  {isSelf ? <span className="text-muted-foreground"> (you)</span> : null}
                </TableCell>
                <TableCell className="text-muted-foreground">{row.name || "-"}</TableCell>
                <TableCell>
                  <Badge variant={row.role === "admin" ? "default" : "muted"}>{row.role}</Badge>
                </TableCell>
                <TableCell className="text-muted-foreground" title={new Date(row.created_at).toLocaleString()}>
                  {formatRelative(row.created_at)}
                </TableCell>
                <TableCell className="text-right">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button type="button" size="icon" variant="ghost" aria-label={`Actions for ${row.email}`}>
                        <MoreHorizontal className="size-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent>
                      <DropdownMenuItem onSelect={() => onEdit(row)}>Edit</DropdownMenuItem>
                      {!isSelf ? (
                        <DropdownMenuItem
                          className="text-destructive focus:text-destructive"
                          onSelect={() => onDelete(row)}
                        >
                          Delete
                        </DropdownMenuItem>
                      ) : null}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
