import { revalidateLogic, useForm } from "@tanstack/react-form";
import { useSearch } from "@tanstack/react-router";
import { z } from "zod";

import { WoodstarMark } from "@/components/brand/woodstar-mark";
import { FormField } from "@/components/form-field";
import { PendingButton } from "@/components/pending-button";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldError, FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { useLogin, useSession } from "@/hooks/use-auth";
import { emailAddress, requiredString } from "@/lib/form-validation";
export function LoginPage() {
  const { session } = useSession();
  const search: {
    sso_error?: string;
  } = useSearch({ strict: false });
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
          <WoodstarMark size="md" />
          <CardTitle>Log In to Woodstar</CardTitle>
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
                  <FormField field={field} label="Email" htmlFor="login-email" required>
                    {(control) => (
                      <Input
                        {...control}
                        type="email"
                        autoComplete="email"
                        placeholder="admin@example.com"
                        required
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </FormField>
                )}
              </form.Field>
              <form.Field name="password">
                {(field) => (
                  <FormField field={field} label="Password" htmlFor="login-password" required>
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
                  </FormField>
                )}
              </form.Field>

              <Field>
                <form.Subscribe selector={(state) => state.isSubmitting}>
                  {(isSubmitting) => (
                    <PendingButton isPending={isSubmitting} type="submit">
                      Login
                    </PendingButton>
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
