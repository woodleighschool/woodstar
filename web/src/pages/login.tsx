import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter, useSearch } from "@tanstack/react-router";

import { WoodstarMark } from "@/components/brand/woodstar-mark";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { useSession } from "@/hooks/use-auth";
import type { ApiError, LoginInput, User } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { formString } from "@/lib/utils";

export function LoginPage() {
  const queryClient = useQueryClient();
  const router = useRouter();
  const { session } = useSession();
  const search: { sso_error?: string } = useSearch({ strict: false });
  const ssoEnabled = session?.sso_enabled ?? false;

  const login = useMutation<User, ApiError, LoginInput>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/auth/login", { body })),
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
          <CardTitle>Log In to Woodstar</CardTitle>
        </CardHeader>
        <CardContent>
          <form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              const form = new FormData(event.currentTarget);
              login.mutate({
                email: formString(form, "email").trim(),
                password: formString(form, "password"),
              });
            }}
          >
            <FieldGroup className="gap-4">
              <Field>
                <FieldLabel htmlFor="login-email" required>
                  Email
                </FieldLabel>
                <Input
                  id="login-email"
                  name="email"
                  type="email"
                  autoComplete="email"
                  placeholder="admin@example.com"
                  required
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="login-password" required>
                  Password
                </FieldLabel>
                <Input id="login-password" name="password" type="password" autoComplete="current-password" required />
              </Field>

              {search.sso_error ? <FieldError>{search.sso_error}</FieldError> : null}

              <Field>
                <Button type="submit" disabled={login.isPending}>
                  {login.isPending ? "Signing in..." : "Login"}
                </Button>

                {ssoEnabled ? (
                  <>
                    <Separator />
                    <Button asChild type="button" variant="outline">
                      <a href="/api/auth/sso/start">Login with SSO</a>
                    </Button>
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
