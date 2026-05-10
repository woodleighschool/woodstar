import { Loader2, MoreHorizontal, UserPlus, Users } from "lucide-react";
import { useState } from "react";

import { PageActions } from "@/components/layout/page-actions";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
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
    <>
      <PageActions>
        <Button size="sm" className="gap-2" onClick={() => setCreateOpen(true)}>
          <UserPlus data-icon="inline-start" /> Add user
        </Button>
      </PageActions>

      <div className="p-6">
        <UsersTable query={query} currentUserId={currentUser?.id ?? null} onEdit={setEditing} onDelete={setDeleting} />
      </div>

      <UserFormDialog mode="create" open={createOpen} onOpenChange={setCreateOpen} />

      {editing ? (
        <UserFormDialog
          mode="edit"
          open
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
    </>
  );
}

interface UsersTableProps {
  query: ReturnType<typeof useUsers>;
  currentUserId: number | null;
  onEdit: (user: User) => void;
  onDelete: (user: User) => void;
}

function UsersTable({ query, currentUserId, onEdit, onDelete }: UsersTableProps) {
  if (query.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to load users</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
        <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
          Retry
        </Button>
      </Alert>
    );
  }

  if (query.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading...
      </div>
    );
  }

  const data = query.data ?? [];
  if (data.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Users />
          </EmptyMedia>
          <EmptyTitle>No users yet</EmptyTitle>
          <EmptyDescription>
            Add a user to give other admins or viewers access to this Woodstar deployment.
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
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
                  <Badge variant={row.role === "admin" ? "default" : "secondary"}>{row.role}</Badge>
                </TableCell>
                <TableCell className="text-muted-foreground" title={new Date(row.created_at).toLocaleString()}>
                  {formatRelative(row.created_at)}
                </TableCell>
                <TableCell className="text-right">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button type="button" size="icon" variant="ghost" aria-label={`Actions for ${row.email}`}>
                        <MoreHorizontal />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuGroup>
                        <DropdownMenuItem onSelect={() => onEdit(row)}>Edit</DropdownMenuItem>
                        {!isSelf ? (
                          <DropdownMenuItem variant="destructive" onSelect={() => onDelete(row)}>
                            Delete
                          </DropdownMenuItem>
                        ) : null}
                      </DropdownMenuGroup>
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
