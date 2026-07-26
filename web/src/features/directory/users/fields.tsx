import { revalidateLogic, useForm } from "@tanstack/react-form";
import type { ReactNode } from "react";
import { z } from "zod";

import { EnumBadge } from "@components/enum-badge";
import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
  FieldTitle,
} from "@components/ui/field";
import { Input } from "@components/ui/input";
import { RadioGroup, RadioGroupItem } from "@components/ui/radio-group";
import { ValidatedFormField } from "@components/validated-form-field";
import { DIRECTORY_SOURCES } from "@features/directory/source";
import {
  USER_ACCESS_ROLE_OPTIONS,
  USER_ACCESS_ROLES,
  USER_ACCESS_ROLE_VALUES,
  userAccessRole,
  type UserAccessRole,
  userMutationRole,
} from "@features/directory/users/metadata";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { User, UserMutation } from "@lib/api";
import { isOneOf } from "@lib/utils";
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
  actions,
  onSubmit,
  onCancel,
}: {
  user: User;
  initial: UserFormState;
  actions?: ReactNode;
  onSubmit: (body: UserMutation) => Promise<void>;
  onCancel: () => void;
}) {
  const isLocal = user.source === "local";
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({
      mode: "submit",
      modeAfterSubmission: "change",
    }),
    validators: { onDynamic: userFormSchema },
    onSubmit: async ({ value, formApi }) => {
      await onSubmit({
        name: isLocal ? value.name.trim() : user.name,
        role: userMutationRole(value.role),
        password: isLocal && value.password.trim() !== "" ? value.password : undefined,
      });
      // Re-baseline so the saved values count as unchanged.
      formApi.reset({ ...value, password: "" });
    },
  });
  const exitGuard = usePageFormExitGuard({
    form,
    onDiscard: onCancel,
  });
  return (
    <>
      <PageShell
        render={
          <form
            noValidate
            onSubmit={(event) => {
              event.preventDefault();
              void form.handleSubmit();
            }}
          />
        }
      >
        <PageHeader
          title="Edit User"
          context={<EnumBadge value={user.source} metadata={DIRECTORY_SOURCES} />}
          actions={actions}
        />

        <FieldGroup className="max-w-3xl">
          <Field>
            <FieldLabel htmlFor="user-email">Email</FieldLabel>
            <Input id="user-email" type="email" value={user.email} disabled />
          </Field>

          <form.Field name="name">
            {(field) => (
              <ValidatedFormField field={field} label="Display Name" htmlFor="user-name">
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
              </ValidatedFormField>
            )}
          </form.Field>

          <form.Field name="role">
            {(field) => (
              <FieldSet>
                <FieldLegend variant="label">Role</FieldLegend>
                <RadioGroup
                  name={field.name}
                  value={field.state.value}
                  className="grid gap-2 md:grid-cols-3"
                  onValueChange={(value) => {
                    if (isOneOf(value, USER_ACCESS_ROLE_VALUES)) field.handleChange(value);
                  }}
                >
                  {USER_ACCESS_ROLE_OPTIONS.map((option) => (
                    <FieldLabel key={option.value} htmlFor={`user-role-${option.value}`}>
                      <Field orientation="horizontal">
                        <RadioGroupItem id={`user-role-${option.value}`} value={option.value} />
                        <FieldContent>
                          <FieldTitle>{option.label}</FieldTitle>
                          <FieldDescription>
                            {USER_ACCESS_ROLES[option.value].description}
                          </FieldDescription>
                        </FieldContent>
                      </Field>
                    </FieldLabel>
                  ))}
                </RadioGroup>
              </FieldSet>
            )}
          </form.Field>

          {isLocal ? (
            <form.Field name="password">
              {(field) => (
                <ValidatedFormField
                  field={field}
                  label="Password"
                  htmlFor="user-password"
                  description="Set a new password."
                >
                  {(control) => (
                    <Input
                      {...control}
                      type="password"
                      autoComplete="new-password"
                      minLength={12}
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  )}
                </ValidatedFormField>
              )}
            </form.Field>
          ) : null}
        </FieldGroup>

        <FormActions form={form} submitLabel="Save" onCancel={exitGuard.requestDiscard} />

        {exitGuard.dialog}
      </PageShell>
    </>
  );
}
