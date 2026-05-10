import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { Star } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ApiError } from "@/lib/api";
import { apiClient, unwrap, type Schemas } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { formString } from "@/lib/utils";

type SetupInput = Schemas["SetupInputBody"];
type UserBody = Schemas["UserBody"];

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
            Create the first administrator account. Subsequent accounts can be invited or provisioned via OIDC after
            sign-in.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="space-y-4"
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
            <div className="space-y-1.5">
              <Label htmlFor="setup-email">Email</Label>
              <Input
                id="setup-email"
                name="email"
                type="email"
                autoComplete="email"
                placeholder="admin@example.com"
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="setup-name">Display name</Label>
              <Input id="setup-name" name="name" autoComplete="name" placeholder="Site administrator" required />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="setup-password">Password</Label>
              <Input
                id="setup-password"
                name="password"
                type="password"
                autoComplete="new-password"
                placeholder="Min 12 characters"
                minLength={12}
                required
              />
            </div>

            {error ? <p className="text-sm text-destructive">{error}</p> : null}

            <Button type="submit" className="w-full" disabled={setup.isPending}>
              {setup.isPending ? "Creating account..." : "Create admin account"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
