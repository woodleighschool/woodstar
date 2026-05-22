import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter, useSearch } from "@tanstack/react-router";
import { useState } from "react";

import { WoodstarMark } from "@/components/brand/woodstar-mark";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { useSession } from "@/hooks/use-auth";
import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { formString } from "@/lib/utils";

type LoginInput = Schemas["LoginInputBody"];
type UserBody = Schemas["User"];

export function LoginPage() {
  const queryClient = useQueryClient();
  const router = useRouter();
  const { session } = useSession();
  const search: { sso_error?: string } = useSearch({ strict: false });
  const [error, setError] = useState<string | null>(search.sso_error ?? null);

  const ssoEnabled = session?.sso_enabled ?? false;

  const login = useMutation<UserBody, ApiError, LoginInput>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/auth/login", { body })),
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
    <div className="flex min-h-dvh w-full min-w-0 items-center justify-center overflow-x-hidden bg-muted/40 px-4 py-10">
      <Card className="w-full max-w-md overflow-hidden">
        <CardHeader className="items-center justify-items-center text-center">
          <WoodstarMark size="md" />
          <CardTitle>Sign in to Woodstar</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              const form = new FormData(event.currentTarget);
              login.mutate({
                email: formString(form, "email"),
                password: formString(form, "password"),
              });
            }}
          >
            <FieldGroup className="gap-4">
              <Field>
                <FieldLabel htmlFor="login-email">Email</FieldLabel>
                <Input id="login-email" name="email" type="email" autoComplete="email" required />
              </Field>
              <Field>
                <FieldLabel htmlFor="login-password">Password</FieldLabel>
                <Input id="login-password" name="password" type="password" autoComplete="current-password" required />
              </Field>
            </FieldGroup>

            <FieldError>{error}</FieldError>

            <Button type="submit" className="w-full" disabled={login.isPending}>
              {login.isPending ? "Signing in..." : "Sign in"}
            </Button>
          </form>

          {ssoEnabled ? (
            <>
              <Separator />
              <Button asChild type="button" variant="outline" className="w-full">
                <a href="/api/auth/sso/start">Sign in with SSO</a>
              </Button>
            </>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
