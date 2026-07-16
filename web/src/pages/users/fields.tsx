import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import { EnumBadge } from "@/components/enum-badge";
import { FormField } from "@/components/form-field";
import { Pending } from "@/components/pending";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useFormExitGuard } from "@/hooks/use-form-exit-guard";
import type { User, UserMutation } from "@/lib/api";
import { directorySourceLabel } from "@/lib/directory";
import {
  USER_ACCESS_ROLE_OPTIONS,
  USER_ACCESS_ROLES,
  userAccessRole,
  type UserAccessRole,
  userMutationRole,
} from "@/lib/users";
import { formatRelative, nonEmpty } from "@/lib/utils";

interface UserFormState {
  name: string;
  role: UserAccessRole;
  password: string;
}

export function userFromDetail(user: User): UserFormState {
  return {
    name: user.name,
    role: userAccessRole(user.role),
    password: "",
  };
}

const userFormSchema = z.object({
  name: z.string(),
  role: z.enum(["admin", "viewer", "none"]),
  password: z
    .string()
    .refine(
      (value) => value.trim() === "" || value.length >= 12,
      "Password must be at least 12 characters.",
    ),
});

export function UserForm({
  user,
  initial,
  onSubmit,
  onSuccess,
}: {
  user: User;
  initial: UserFormState;
  onSubmit: (body: UserMutation) => Promise<number | undefined>;
  onSuccess?: (id: number | undefined) => void;
}) {
  const isLocal = user.source === "local";

  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: userFormSchema },
    onSubmit: async ({ value, formApi }) => {
      const id = await onSubmit({
        name: isLocal ? value.name.trim() : user.name,
        role: userMutationRole(value.role),
        password: isLocal && value.password.trim() !== "" ? value.password : undefined,
      });
      // Re-baseline so the saved values count as unchanged.
      formApi.reset({ ...value, password: "" });
      onSuccess?.(id);
    },
  });
  const exitGuard = useFormExitGuard({
    form,
    onDiscard: () => form.reset(initial),
  });

  return (
    <>
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
                    description={
                      !isLocal ? `Managed by ${directorySourceLabel(user.source)}.` : undefined
                    }
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
          <CardFooter className="flex items-center justify-between gap-3 pt-6">
            <p
              className="text-xs text-muted-foreground"
              title={new Date(user.updated_at).toLocaleString()}
            >
              Updated {formatRelative(user.updated_at)}
            </p>
            <form.Subscribe selector={(state) => state.isSubmitting}>
              {(isSubmitting) => (
                <Pending isPending={isSubmitting}>
                  <Button type="submit" size="sm">
                    {isSubmitting ? "Saving…" : "Save"}
                  </Button>
                </Pending>
              )}
            </form.Subscribe>
          </CardFooter>
        </form>
      </Card>
      {exitGuard.dialog}
    </>
  );
}
