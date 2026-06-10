import { useForm } from "@tanstack/react-form";
import { useNavigate, useParams } from "@tanstack/react-router";
import { Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { z } from "zod";

import { EnumBadge } from "@/components/enum-badge";
import { FormField } from "@/components/form-field";
import { PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { SubmitButton } from "@/components/submit-button";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { UserDeleteDialog } from "@/components/users/user-delete-dialog";
import { useAuth } from "@/hooks/use-auth";
import { useUpdateUser, useUser, type User } from "@/hooks/use-users";
import { directorySourceLabel } from "@/lib/directory";
import {
  USER_ACCESS_ROLES,
  USER_ACCESS_ROLE_OPTIONS,
  userAccessRole,
  userMutationRole,
  type UserAccessRole,
} from "@/lib/users";
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
        <QueryError title="Failed to load user" error={user.error} onRetry={() => void user.refetch()} />
      </PageShell>
    );
  }

  if (!user.data) {
    return null;
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
  const isLocal = user.source === "local";

  const form = useForm({
    defaultValues: {
      name: user.name,
      role: userAccessRole(user.role),
      password: "",
    },
    validators: {
      onSubmit: z.object({
        name: z.string(),
        role: z.enum(["admin", "viewer", "none"]),
        password: z.string(),
      }),
    },
    onSubmit: async ({ value }) => {
      const saved = await update.mutateAsync({
        id: user.id,
        body: {
          name: isLocal ? value.name.trim() : user.name,
          role: userMutationRole(value.role),
          password: isLocal && value.password.trim() !== "" ? value.password : undefined,
        },
      });
      form.setFieldValue("password", "");
      toast.success("User updated");
      void navigate({ to: "/directory/users/$userId/edit", params: { userId: String(saved.id) } });
    },
  });

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
          noValidate
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit();
          }}
        >
          <CardContent>
            <FieldGroup className="gap-4">
              <form.Field name="name">
                {(field) => (
                  <FormField
                    field={field}
                    label="Display Name"
                    htmlFor="user-name"
                    description={!isLocal ? `Managed by ${directorySourceLabel(user.source)}.` : undefined}
                  >
                    {(control) => (
                      <Input
                        {...control}
                        type="text"
                        autoComplete="off"
                        disabled={!isLocal}
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </FormField>
                )}
              </form.Field>

              <form.Field name="role">
                {(field) => (
                  <FormField field={field} label="Role" htmlFor="user-role">
                    {(control) => (
                      <Select
                        value={field.state.value}
                        onValueChange={(value) => field.handleChange(value as UserAccessRole)}
                      >
                        <SelectTrigger {...control} className="w-full">
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
                    )}
                  </FormField>
                )}
              </form.Field>

              <form.Field name="password">
                {(field) => (
                  <FormField
                    field={field}
                    label="Password"
                    htmlFor="user-password"
                    description={
                      isLocal
                        ? "Set a new password."
                        : `${directorySourceLabel(user.source)} users do not use local passwords.`
                    }
                  >
                    {(control) => (
                      <Input
                        {...control}
                        type="password"
                        autoComplete="new-password"
                        minLength={12}
                        disabled={!isLocal}
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </FormField>
                )}
              </form.Field>
            </FieldGroup>
          </CardContent>
          <CardFooter className="flex justify-between gap-3 pt-6">
            <p className="text-muted-foreground text-xs" title={new Date(user.updated_at).toLocaleString()}>
              Updated {formatRelative(user.updated_at)}
            </p>
            <form.Subscribe selector={(state) => state.values}>
              {(values) => {
                const changed =
                  (isLocal && values.name !== user.name) ||
                  userMutationRole(values.role) !== user.role ||
                  (isLocal && values.password.trim() !== "");
                return (
                  <SubmitButton pending={update.isPending} disabled={!changed} size="sm">
                    Save
                  </SubmitButton>
                );
              }}
            </form.Subscribe>
          </CardFooter>
        </form>
      </Card>

      <Card className="gap-4 py-4">
        <CardHeader className="px-4">
          <CardTitle>Delete User</CardTitle>
        </CardHeader>
        <CardContent className="px-4">
          <Button type="button" variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
            <Trash2 data-icon="inline-start" />
            Delete
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
