import { revalidateLogic, useForm } from "@tanstack/react-form";
import { toast } from "sonner";
import { z } from "zod";

import { AsyncButton } from "@components/async-button";
import { EnumBadge } from "@components/enum-badge";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@components/ui/card";
import { FieldGroup } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { ValidatedFormField } from "@components/validated-form-field";
import { APIKeyCard } from "@features/account/api-key-card";
import { useAccount, useUpdateAccount } from "@features/account/queries";
import { directorySourceLabel } from "@features/directory/source";
import { USER_ACCESS_ROLES, userAccessRole } from "@features/directory/users/metadata";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { Account } from "@lib/api";
import { formatRelative } from "@lib/utils";
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
  return (
    <PageShell className="max-w-3xl gap-4">
      <PageHeader title="Account" />
      <AccountProfileCard key={account.data.user.updated_at} account={account.data} />
      <APIKeyCard />
    </PageShell>
  );
}
function AccountProfileCard({ account }: { account: Account }) {
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
      toast.success("Account saved");
    },
  });
  const exitGuard = usePageFormExitGuard({
    form,
    onDiscard: () => form.reset(initial),
  });
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Profile</CardTitle>
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
                  <ValidatedFormField
                    field={field}
                    label="Display Name"
                    htmlFor="account-name"
                    description={
                      !isLocal ? `Managed by ${directorySourceLabel(user.source)}.` : undefined
                    }
                  >
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

              <form.Field name="password">
                {(field) => (
                  <ValidatedFormField
                    field={field}
                    label="Password"
                    htmlFor="account-password"
                    description={
                      isLocal
                        ? "Set a new password."
                        : `${directorySourceLabel(user.source)} accounts do not use local passwords.`
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
                  </ValidatedFormField>
                )}
              </form.Field>
            </FieldGroup>
          </CardContent>
          <CardFooter className="flex justify-between gap-3 pt-6">
            <p
              className="text-xs text-muted-foreground"
              title={new Date(user.updated_at).toLocaleString()}
            >
              Updated {formatRelative(user.updated_at)}
            </p>
            <form.Subscribe selector={(state) => state.isSubmitting}>
              {(isSubmitting) => (
                <AsyncButton isPending={isSubmitting} type="submit" size="sm">
                  Save
                </AsyncButton>
              )}
            </form.Subscribe>
          </CardFooter>
        </form>
      </Card>
      {exitGuard.dialog}
    </>
  );
}
