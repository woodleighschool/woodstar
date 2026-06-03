import { useNavigate, useParams } from "@tanstack/react-router";
import { Loader2, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { EnumBadge } from "@/components/enum-badge";
import { PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { UserDeleteDialog } from "@/components/users/user-delete-dialog";
import {
  USER_ACCESS_ROLES,
  USER_ACCESS_ROLE_OPTIONS,
  userAccessRole,
  userMutationRole,
  type UserAccessRole,
} from "@/components/users/user-role";
import { useAuth } from "@/hooks/use-auth";
import { useUpdateUser, useUser, type User } from "@/hooks/use-users";
import { formatRelative, nonEmpty } from "@/lib/utils";
import { AccountPage } from "@/pages/account";

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
          <AlertTitle>Failed to Load User</AlertTitle>
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
  const [role, setRole] = useState<UserAccessRole>(userAccessRole(user.role));
  const [password, setPassword] = useState("");

  const passwordChanged = password.trim() !== "";
  const nameChanged = !user.synced && name !== user.name;
  const roleChanged = userMutationRole(role) !== user.role;
  const changed = nameChanged || roleChanged || passwordChanged;

  async function submit() {
    if (!changed) return;
    const saved = await update.mutateAsync({
      id: user.id,
      body: {
        name: user.synced ? user.name : name.trim(),
        role: userMutationRole(role),
        password: passwordChanged ? password : undefined,
      },
    });
    setPassword("");
    toast.success("User updated");
    void navigate({ to: "/directory/users/$userId/edit", params: { userId: String(saved.id) } });
  }

  return (
    <PageShell className="max-w-3xl gap-4">
      <Card>
        <CardHeader>
          <CardTitle>{nonEmpty(user.name) ?? user.email}</CardTitle>
          <CardDescription className="flex flex-wrap items-center gap-2">
            <span>{user.email}</span>
            <EnumBadge value={userAccessRole(user.role)} metadata={USER_ACCESS_ROLES} />
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
              <Field data-disabled={user.synced}>
                <FieldLabel htmlFor="user-name">Display Name</FieldLabel>
                <Input
                  id="user-name"
                  type="text"
                  autoComplete="off"
                  value={name}
                  disabled={user.synced}
                  onChange={(event) => setName(event.target.value)}
                />
                {user.synced ? <FieldDescription>Synced from Entra.</FieldDescription> : null}
              </Field>

              <Field>
                <FieldLabel htmlFor="user-role">Role</FieldLabel>
                <Select value={role} onValueChange={(value) => setRole(value as UserAccessRole)}>
                  <SelectTrigger id="user-role" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {USER_ACCESS_ROLE_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>

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

      <Card className="gap-4 py-4">
        <CardHeader className="px-4">
          <CardTitle>Remove User</CardTitle>
        </CardHeader>
        <CardContent className="px-4">
          <Button type="button" variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
            <Trash2 data-icon="inline-start" />
            {user.synced ? "Deactivate User" : "Delete User"}
          </Button>
        </CardContent>
      </Card>

      <UserDeleteDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        user={user}
        onDeleted={() => void navigate({ to: "/directory/users" })}
      />
    </PageShell>
  );
}
