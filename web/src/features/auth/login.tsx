import { revalidateLogic, useForm } from "@tanstack/react-form";
import { getRouteApi } from "@tanstack/react-router";
import { z } from "zod";

import { AsyncButton } from "@components/async-button";
import { Logo } from "@components/logo";
import { Button } from "@components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@components/ui/card";
import { Field, FieldError, FieldGroup } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { Separator } from "@components/ui/separator";
import { ValidatedFormField } from "@components/validated-form-field";
import { useLogin, useSession } from "@features/auth/queries";
import { emailAddress, requiredString } from "@lib/form-validation";

const routeApi = getRouteApi("/login");

export function LoginPage() {
  const { session } = useSession();
  const search = routeApi.useSearch();
  const ssoEnabled = session?.sso_enabled ?? false;
  const login = useLogin();
  const form = useForm({
    defaultValues: { email: "", password: "" },
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: {
      onDynamic: z.object({
        email: emailAddress(),
        password: requiredString("Password"),
      }),
    },
    onSubmit: async ({ value }) => {
      await login.mutateAsync({ email: value.email.trim(), password: value.password });
    },
  });
  return (
    <div className="flex min-h-dvh w-full min-w-0 items-center justify-center overflow-x-hidden bg-muted/40 px-4 py-10">
      <Card className="w-full max-w-md">
        <CardHeader className="items-center justify-items-center text-center">
          <Logo size="md" />
          <CardTitle>Log In</CardTitle>
        </CardHeader>
        <CardContent>
          <form
            noValidate
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              void form.handleSubmit();
            }}
          >
            <FieldGroup className="gap-4">
              <form.Field name="email">
                {(field) => (
                  <ValidatedFormField field={field} label="Email" htmlFor="login-email" required>
                    {(control) => (
                      <Input
                        {...control}
                        type="email"
                        autoComplete="email"
                        required
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
                    htmlFor="login-password"
                    required
                  >
                    {(control) => (
                      <Input
                        {...control}
                        type="password"
                        autoComplete="current-password"
                        required
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </ValidatedFormField>
                )}
              </form.Field>

              <Field>
                <form.Subscribe selector={(state) => state.isSubmitting}>
                  {(isSubmitting) => (
                    <AsyncButton isPending={isSubmitting} type="submit">
                      Login
                    </AsyncButton>
                  )}
                </form.Subscribe>

                {search.sso_error || login.error ? (
                  <FieldError>{search.sso_error ?? login.error?.message}</FieldError>
                ) : null}

                {ssoEnabled ? (
                  <>
                    <Separator />
                    <Button
                      type="button"
                      variant="outline"
                      render={<a href="/api/auth/sso/start">Login with SSO</a>}
                      nativeButton={false}
                    />
                  </>
                ) : null}
              </Field>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
