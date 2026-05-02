import { Star } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { endpoints } from "@/lib/endpoints";

export function SetupPage() {
  const disabled = !endpoints.setupComplete.implemented;

  return (
    <div className="flex min-h-dvh items-center justify-center bg-muted/40 px-4 py-10">
      <Card className="w-full max-w-md">
        <CardHeader className="items-center text-center">
          <div className="rounded-full bg-primary/10 p-2 text-primary">
            <Star className="size-5" />
          </div>
          <CardTitle>Set up Woodstar</CardTitle>
          <CardDescription>
            Create the first administrator account. Subsequent accounts can be
            invited or provisioned via OIDC after sign-in.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="space-y-4"
            onSubmit={(e) => {
              e.preventDefault();
            }}
          >
            <div className="space-y-1.5">
              <Label htmlFor="setup-email">Email</Label>
              <Input
                id="setup-email"
                type="email"
                autoComplete="email"
                placeholder="admin@example.com"
                disabled={disabled}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="setup-name">Display name</Label>
              <Input
                id="setup-name"
                autoComplete="name"
                placeholder="Site administrator"
                disabled={disabled}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="setup-password">Password</Label>
              <Input
                id="setup-password"
                type="password"
                autoComplete="new-password"
                placeholder="Min 12 characters"
                minLength={12}
                disabled={disabled}
                required
              />
            </div>

            <Button type="submit" className="w-full" disabled={disabled}>
              {disabled ? "Setup endpoint pending" : "Create admin account"}
            </Button>

            {disabled ? (
              <p className="text-xs text-muted-foreground text-center">
                The setup endpoint hasn&apos;t been wired yet. This form is a
                shell so the first-run flow is reviewable.
              </p>
            ) : null}
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
