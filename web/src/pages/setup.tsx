import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { Star } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
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
  const [error, setError] = useState<string | null>(null);

  const setup = useMutation<UserBody, ApiError, SetupInput>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/setup", { body })),
    onSuccess: async () => {
      setError(null);
      await queryClient.invalidateQueries({ queryKey: queryKeys.session });
      await router.navigate({ to: "/hosts" });
    },
    onError: (err) => {
      setError(err.message);
    },
  });

  return (
    <div className="flex min-h-dvh items-center justify-center bg-muted/40 px-4 py-10">
      <Card className="w-full max-w-md">
        <CardHeader className="items-center text-center">
          <div className="rounded-full bg-primary/10 p-2 text-primary">
            <Star className="size-5" />
          </div>
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
                email: formString(form, "email"),
                name: formString(form, "name"),
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

            <FieldError>{error}</FieldError>

            <Button type="submit" className="w-full" disabled={setup.isPending}>
              {setup.isPending ? "Creating account..." : "Create admin account"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
