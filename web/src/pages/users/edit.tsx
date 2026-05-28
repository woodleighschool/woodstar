import { useNavigate, useParams } from "@tanstack/react-router";
import { Loader2, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { UserDeleteDialog } from "@/components/users/user-delete-dialog";
import { useAuth } from "@/hooks/use-auth";
import { useUpdateUser, useUser, type User } from "@/hooks/use-users";
import { formatRelative, nonEmpty } from "@/lib/utils";
import { AccountPage } from "@/pages/account";

const INITIAL_USER_ID = 1;

export function UserEditPage() {
  const params = useParams({ strict: false });
  const userId = params.userId ?? "";
  const userID = Number(userId);
  const user = useUser(userID);
  const { user: currentUser } = useAuth();

  if (user.error) {
    return (
      <PageShell>
        <Alert variant="destructive">
          <AlertTitle>Failed to load user</AlertTitle>
          <AlertDescription>{user.error.message}</AlertDescription>
          <Button variant="outline" size="sm" onClick={() => void user.refetch()} className="mt-2 w-fit">
            Retry
          </Button>
        </Alert>
      </PageShell>
    );
  }

  if (!user.data) {
    return (
      <PageShell className="max-w-3xl gap-4">
        <Skeleton className="h-36" />
        <Skeleton className="h-28" />
      </PageShell>
    );
  }

  if (currentUser?.id === user.data.id) {
    return <AccountPage />;
  }

  return <UserEditForm key={user.data.updated_at} user={user.data} />;
}

function UserEditForm({ user }: { user: User }) {
  const navigate = useNavigate();
  const update = useUpdateUser();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [name, setName] = useState(user.name);
  const [role, setRole] = useState<User["role"]>(user.role);
  const [password, setPassword] = useState("");

  const isInitialUser = user.id === INITIAL_USER_ID;
  const canDelete = !isInitialUser;
  const passwordChanged = password.trim() !== "";
  const changed = (!isInitialUser && (name !== user.name || role !== user.role)) || passwordChanged;

  async function submit() {
    if (!changed) return;
    const saved = await update.mutateAsync({
      id: user.id,
      body: {
        name: isInitialUser ? user.name : name.trim(),
        role: isInitialUser ? user.role : role,
        password: passwordChanged ? password : undefined,
      },
    });
    setPassword("");
    toast.success("User updated");
    void navigate({ to: "/users/$userId/edit", params: { userId: String(saved.id) } });
  }

  return (
    <PageShell className="max-w-3xl gap-4">
      <Card>
        <CardHeader>
          <CardTitle>{nonEmpty(user.name) ?? user.email}</CardTitle>
          <CardDescription className="flex flex-wrap items-center gap-2">
            <span>{user.email}</span>
            <Badge variant={user.role === "admin" ? "default" : "secondary"}>{user.role}</Badge>
          </CardDescription>
        </CardHeader>
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          <CardContent>
            <FieldGroup className="gap-4">
              {!isInitialUser ? (
                <>
                  <Field>
                    <FieldLabel htmlFor="user-name">Display name</FieldLabel>
                    <Input
                      id="user-name"
                      type="text"
                      autoComplete="off"
                      value={name}
                      onChange={(event) => setName(event.target.value)}
                    />
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="user-role">Role</FieldLabel>
                    <Select value={role} onValueChange={(value) => setRole(value as User["role"])}>
                      <SelectTrigger id="user-role" className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          <SelectItem value="admin">admin</SelectItem>
                          <SelectItem value="viewer">viewer</SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  </Field>
                </>
              ) : null}

              <Field>
                <FieldLabel htmlFor="user-password">Password</FieldLabel>
                <Input
                  id="user-password"
                  type="password"
                  autoComplete="new-password"
                  minLength={12}
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
                <FieldDescription>Set a new password.</FieldDescription>
              </Field>
            </FieldGroup>
          </CardContent>
          <CardFooter className="flex justify-between gap-3 pt-6">
            <p className="text-muted-foreground text-xs" title={new Date(user.updated_at).toLocaleString()}>
              Updated {formatRelative(user.updated_at)}
            </p>
            <Button type="submit" size="sm" disabled={!changed || update.isPending}>
              {update.isPending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
              Save
            </Button>
          </CardFooter>
        </form>
      </Card>

      {canDelete ? (
        <Card className="gap-4 py-4">
          <CardHeader className="px-4">
            <CardTitle>Remove user</CardTitle>
          </CardHeader>
          <CardContent className="px-4">
            <Button type="button" variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
              <Trash2 data-icon="inline-start" />
              Delete user
            </Button>
          </CardContent>
        </Card>
      ) : null}

      <UserDeleteDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        user={canDelete ? user : null}
        onDeleted={() => void navigate({ to: "/users" })}
      />
    </PageShell>
  );
}
