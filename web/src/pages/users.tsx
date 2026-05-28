import { Link } from "@tanstack/react-router";
import { Loader2, MoreHorizontal, UserPlus, Users } from "lucide-react";
import { useState } from "react";

import { DataTableEmptyState } from "@/components/data-table/data-table-empty-state";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { UserDeleteDialog } from "@/components/users/user-delete-dialog";
import { UserFormDialog } from "@/components/users/user-form-dialog";
import { useAuth } from "@/hooks/use-auth";
import { useUsers, type User } from "@/hooks/use-users";
import { formatRelative } from "@/lib/utils";

const INITIAL_USER_ID = 1;

export function UsersPage() {
  const query = useUsers();
  const { user: currentUser } = useAuth();

  const [createOpen, setCreateOpen] = useState(false);
  const [deleting, setDeleting] = useState<User | null>(null);

  return (
    <PageShell>
      <PageHeader
        title="Users"
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
    return <DataTableEmptyState icon={<Users />} title="No account access" description="Create a local account." />;
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Created</TableHead>
            <TableHead className="w-12" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row) => {
            const isSelf = row.id === currentUserId;
            const isInitial = row.id === INITIAL_USER_ID;
            return (
              <TableRow key={row.id}>
                <TableCell className="font-medium">{row.name || row.email}</TableCell>
                <TableCell>
                  {row.email}
                  {isSelf ? <span className="text-muted-foreground"> (you)</span> : null}
                  {isInitial ? <span className="text-muted-foreground"> (initial)</span> : null}
                </TableCell>
                <TableCell>
                  <Badge variant={row.role === "admin" ? "info" : "secondary"}>{row.role}</Badge>
                </TableCell>
                <TableCell title={new Date(row.created_at).toLocaleString()}>
                  {formatRelative(row.created_at)}
                </TableCell>
                <TableCell className="text-right">
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
                            <Link to="/users/$userId/edit" params={{ userId: String(row.id) }}>
                              Edit
                            </Link>
                          </DropdownMenuItem>
                        )}
                        {!isSelf && !isInitial ? (
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
