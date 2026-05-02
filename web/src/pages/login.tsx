import { Star } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { endpoints } from "@/lib/endpoints";

export function LoginPage() {
  const disabled = !endpoints.authLogin.implemented;

  return (
    <div className="flex min-h-dvh items-center justify-center bg-muted/40 px-4 py-10">
      <Card className="w-full max-w-md">
        <CardHeader className="items-center text-center">
          <div className="rounded-full bg-primary/10 p-2 text-primary">
            <Star className="size-5" />
          </div>
          <CardTitle>Sign in to Woodstar</CardTitle>
          <CardDescription>
            Local administrator credentials or OIDC, when configured.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <form
            className="space-y-4"
            onSubmit={(e) => {
              e.preventDefault();
            }}
          >
            <div className="space-y-1.5">
              <Label htmlFor="login-email">Email</Label>
              <Input
                id="login-email"
                type="email"
                autoComplete="email"
                disabled={disabled}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="login-password">Password</Label>
              <Input
                id="login-password"
                type="password"
                autoComplete="current-password"
                disabled={disabled}
                required
              />
            </div>

            <Button type="submit" className="w-full" disabled={disabled}>
              {disabled ? "Login endpoint pending" : "Sign in"}
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

          {disabled ? (
            <p className="text-xs text-muted-foreground text-center">
              Authentication endpoints are not yet wired. This is the login
              shell for review.
            </p>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
