import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import { WoodstarMark } from "@/components/brand/woodstar-mark";
import { FormField } from "@/components/form-field";
import { Pending } from "@/components/pending";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldError, FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { useSetup } from "@/hooks/use-auth";
import { useFormExitGuard } from "@/hooks/use-form-exit-guard";
import { emailAddress, requiredString } from "@/lib/form-validation";

export function SetupPage() {
  const setup = useSetup();
  const initial = { email: "", name: "", password: "" };

  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: {
      onDynamic: z.object({
        email: emailAddress(),
        name: requiredString("Display name"),
        password: z.string().min(12, "Password must be at least 12 characters."),
      }),
    },
    onSubmit: async ({ value }) => {
      await setup.mutateAsync({
        email: value.email.trim(),
        name: value.name.trim(),
        password: value.password,
      });
      form.reset(value);
    },
  });
  const exitGuard = useFormExitGuard({ form, onDiscard: () => form.reset(initial) });

  return (
    <div className="flex min-h-dvh w-full min-w-0 items-center justify-center overflow-x-hidden bg-muted/40 px-4 py-10">
      <Card className="w-full max-w-md">
        <CardHeader className="items-center justify-items-center text-center">
          <WoodstarMark size="md" />
          <CardTitle>Create an Account</CardTitle>
          <CardDescription>First administrator.</CardDescription>
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
                  <FormField
                    field={field}
                    label="Email"
                    htmlFor="setup-email"
                    required
                    description="Break-glass identity."
                  >
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
              <form.Field name="name">
                {(field) => (
                  <FormField field={field} label="Display Name" htmlFor="setup-name" required>
                    {(control) => (
                      <Input
                        {...control}
                        autoComplete="name"
                        placeholder="Site administrator"
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
                  <FormField
                    field={field}
                    label="Password"
                    htmlFor="setup-password"
                    required
                    description="Cannot be deleted later."
                  >
                    {(control) => (
                      <Input
                        {...control}
                        type="password"
                        autoComplete="new-password"
                        placeholder="Min 12 characters"
                        minLength={12}
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
                    <Pending isPending={isSubmitting}>
                      <Button type="submit">
                        {isSubmitting ? "Creating account…" : "Create Account"}
                      </Button>
                    </Pending>
                  )}
                </form.Subscribe>
                {setup.error ? <FieldError>{setup.error.message}</FieldError> : null}
              </Field>
            </FieldGroup>
            {exitGuard.dialog}
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
