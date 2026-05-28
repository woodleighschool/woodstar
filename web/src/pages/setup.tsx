import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";

import { WoodstarMark } from "@/components/brand/woodstar-mark";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { formString } from "@/lib/utils";

type SetupInput = Schemas["SetupInputBody"];
type UserBody = Schemas["User"];

export function SetupPage() {
  const queryClient = useQueryClient();
  const router = useRouter();

  const setup = useMutation<UserBody, ApiError, SetupInput>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/setup", { body })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.session });
      await router.navigate({ to: "/hosts" });
    },
  });

  return (
    <div className="flex min-h-dvh w-full min-w-0 items-center justify-center overflow-x-hidden bg-muted/40 px-4 py-10">
      <Card className="w-full max-w-md">
        <CardHeader className="items-center justify-items-center text-center">
          <WoodstarMark size="md" />
          <CardTitle>Create an account</CardTitle>
          <CardDescription>First administrator.</CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              const form = new FormData(event.currentTarget);
              setup.mutate({
                email: formString(form, "email").trim(),
                name: formString(form, "name").trim(),
                password: formString(form, "password"),
              });
            }}
          >
            <FieldGroup className="gap-4">
              <Field>
                <FieldLabel htmlFor="setup-email">Email</FieldLabel>
                <Input
                  id="setup-email"
                  name="email"
                  type="email"
                  autoComplete="email"
                  placeholder="admin@example.com"
                  required
                />
                <FieldDescription>Break-glass identity.</FieldDescription>
              </Field>
              <Field>
                <FieldLabel htmlFor="setup-name">Display name</FieldLabel>
                <Input id="setup-name" name="name" autoComplete="name" placeholder="Site administrator" required />
              </Field>
              <Field>
                <FieldLabel htmlFor="setup-password">Password</FieldLabel>
                <Input
                  id="setup-password"
                  name="password"
                  type="password"
                  autoComplete="new-password"
                  placeholder="Min 12 characters"
                  minLength={12}
                  required
                />
                <FieldDescription>Cannot be deleted later.</FieldDescription>
              </Field>

              <Field>
                <Button type="submit" disabled={setup.isPending}>
                  {setup.isPending ? "Creating account..." : "Create account"}
                </Button>
              </Field>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
