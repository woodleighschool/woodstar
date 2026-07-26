import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import { EnumBadge } from "@components/enum-badge";
import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import { Field, FieldGroup, FieldLabel } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { toast } from "@components/ui/toast";
import { ValidatedFormField } from "@components/validated-form-field";
import { APIKeySection } from "@features/account/api-key-section";
import { useAccount, useUpdateAccount } from "@features/account/queries";
import { DIRECTORY_SOURCES } from "@features/directory/source";
import { USER_ACCESS_ROLES, userAccessRole } from "@features/directory/users/metadata";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { Account } from "@lib/api";

export function AccountPage() {
  const account = useAccount();
  if (account.error) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load account"
          error={account.error}
          onRetry={() => void account.refetch()}
        />
      </PageShell>
    );
  }
  if (!account.data) {
    return null;
  }
  return <AccountForm key={account.data.user.updated_at} account={account.data} />;
}

function AccountForm({ account }: { account: Account }) {
  const user = account.user;
  const update = useUpdateAccount();
  const isLocal = user.source === "local";
  const initial = { name: user.name, password: "" };
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: {
      onDynamic: z.object({
        name: z.string(),
        password: z
          .string()
          .refine(
            (value) => value.trim() === "" || value.length >= 12,
            "Password must be at least 12 characters.",
          ),
      }),
    },
    onSubmit: async ({ value }) => {
      await update.mutateAsync({
        name: value.name.trim(),
        password: value.password.trim() !== "" ? value.password : undefined,
      });
      // Re-baseline so the saved values count as unchanged.
      form.reset({ name: value.name, password: "" });
      toast.add({ title: "Account saved", type: "success" });
    },
  });
  const exitGuard = usePageFormExitGuard({
    form,
    onDiscard: () => form.reset(initial),
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
          title="Account"
          context={
            <>
              <EnumBadge value={user.source} metadata={DIRECTORY_SOURCES} />
              <EnumBadge value={userAccessRole(user.role)} metadata={USER_ACCESS_ROLES} />
            </>
          }
        />

        <FieldGroup className="max-w-3xl">
          <Field>
            <FieldLabel htmlFor="account-email">Email</FieldLabel>
            <Input id="account-email" type="email" value={user.email} disabled />
          </Field>

          <form.Field name="name">
            {(field) => (
              <ValidatedFormField field={field} label="Display Name" htmlFor="account-name">
                {(control) => (
                  <Input
                    {...control}
                    type="text"
                    autoComplete="name"
                    disabled={!isLocal}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </ValidatedFormField>
            )}
          </form.Field>

          {isLocal ? (
            <form.Field name="password">
              {(field) => (
                <ValidatedFormField
                  field={field}
                  label="Password"
                  htmlFor="account-password"
                  description="Leave blank to keep the current password."
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

        <APIKeySection account={account} />

        {exitGuard.dialog}
      </PageShell>
    </>
  );
}
