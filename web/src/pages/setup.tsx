import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";

import { WoodstarMark } from "@/components/brand/woodstar-mark";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
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
      <Card className="w-full max-w-md overflow-hidden">
        <CardHeader className="items-center justify-items-center text-center">
          <WoodstarMark size="md" />
          <CardTitle>Set up Woodstar</CardTitle>
          <CardDescription>
            Create the first administrator account. This account is permanent: it cannot be deleted and cannot sign in
            with SSO. Use a generic or break-glass identity so an admin can always log in. Only the password may be
            changed later. Other accounts can be added after sign-in.
          </CardDescription>
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
              </Field>
            </FieldGroup>

            <Button type="submit" className="w-full" disabled={setup.isPending}>
              {setup.isPending ? "Creating account..." : "Create admin account"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
