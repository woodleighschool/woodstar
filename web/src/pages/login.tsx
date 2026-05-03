import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { Star } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { ApiError, apiClient, type Schemas, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

type LoginInput = Schemas["LoginInputBody"];
type UserBody = Schemas["UserBody"];

export function LoginPage() {
  const queryClient = useQueryClient();
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);

  const login = useMutation<UserBody, ApiError, LoginInput>({
    mutationFn: (body) =>
      unwrap(apiClient.POST("/api/auth/login", { body })),
    onSuccess: async (user) => {
      setError(null);
      queryClient.setQueryData(queryKeys.authMe, user);
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
          <CardTitle>Sign in to Woodstar</CardTitle>
          <CardDescription>
            Local administrator sign-in or OIDC, when configured.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <form
            className="space-y-4"
            onSubmit={(e) => {
              e.preventDefault();
              const form = new FormData(e.currentTarget);
              login.mutate({
                email: String(form.get("email") ?? ""),
                password: String(form.get("password") ?? ""),
              });
            }}
          >
            <div className="space-y-1.5">
              <Label htmlFor="login-email">Email</Label>
              <Input
                id="login-email"
                name="email"
                type="email"
                autoComplete="email"
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="login-password">Password</Label>
              <Input
                id="login-password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
              />
            </div>

            {error ? (
              <p className="text-sm text-destructive">{error}</p>
            ) : null}

            <Button type="submit" className="w-full" disabled={login.isPending}>
              {login.isPending ? "Signing in…" : "Sign in"}
            </Button>
          </form>

          <div className="flex items-center gap-3 text-xs uppercase tracking-wider text-muted-foreground">
            <Separator className="flex-1" />
            <span>or</span>
            <Separator className="flex-1" />
          </div>

          <Button variant="outline" className="w-full" disabled>
            Sign in with OIDC
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
