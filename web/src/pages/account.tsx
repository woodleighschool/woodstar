import { Loader2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { APIKeyCard } from "@/components/account/api-key-card";
import { PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useAccount, useUpdateAccount, type Account } from "@/hooks/use-account";
import { formatRelative, nonEmpty } from "@/lib/utils";

const INITIAL_USER_ID = 1;

export function AccountPage() {
  const account = useAccount();

  if (account.error) {
    return (
      <PageShell>
        <Alert variant="destructive">
          <AlertTitle>Failed to load account</AlertTitle>
          <AlertDescription>{account.error.message}</AlertDescription>
          <Button variant="outline" size="sm" onClick={() => void account.refetch()} className="mt-2 w-fit">
            Retry
          </Button>
        </Alert>
      </PageShell>
    );
  }

  if (!account.data) {
    return (
      <PageShell className="max-w-3xl gap-4">
        <Skeleton className="h-32" />
        <Skeleton className="h-28" />
      </PageShell>
    );
  }

  return (
    <PageShell className="max-w-3xl gap-4">
      <AccountProfileCard key={account.data.user.updated_at} account={account.data} />
      <APIKeyCard />
    </PageShell>
  );
}

function AccountProfileCard({ account }: { account: Account }) {
  const user = account.user;
  const isInitialUser = user.id === INITIAL_USER_ID;
  const update = useUpdateAccount();
  const [name, setName] = useState(user.name);
  const [password, setPassword] = useState("");

  const passwordChanged = password.trim() !== "";
  const nameChanged = !isInitialUser && name !== user.name;
  const canSubmit = nameChanged || passwordChanged;

  async function submit() {
    if (!canSubmit) return;
    await update.mutateAsync({
      name: isInitialUser ? user.name : name,
      password: passwordChanged ? password : undefined,
    });
    setPassword("");
    toast.success("Account updated");
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{nonEmpty(user.name) ?? user.email}</CardTitle>
        <CardDescription className="flex flex-wrap items-center gap-2">
          <span>{user.email}</span>
          <Badge variant={user.role === "admin" ? "default" : "secondary"}>{user.role}</Badge>
        </CardDescription>
      </CardHeader>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <CardContent>
          <FieldGroup className="gap-4">
            {!isInitialUser ? (
              <Field>
                <FieldLabel htmlFor="account-name">Display name</FieldLabel>
                <Input
                  id="account-name"
                  type="text"
                  autoComplete="name"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                />
              </Field>
            ) : null}

            <Field>
              <FieldLabel htmlFor="account-password">Password</FieldLabel>
              <Input
                id="account-password"
                type="password"
                autoComplete="new-password"
                minLength={12}
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
              <FieldDescription>Leave blank to keep the current password.</FieldDescription>
            </Field>

            <FieldError>{update.error?.message}</FieldError>
          </FieldGroup>
        </CardContent>
        <CardFooter className="flex justify-between gap-3 pt-6">
          <p className="text-muted-foreground text-xs" title={new Date(user.updated_at).toLocaleString()}>
            Updated {formatRelative(user.updated_at)}
          </p>
          <Button type="submit" size="sm" disabled={!canSubmit || update.isPending}>
            {update.isPending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
            Save changes
          </Button>
        </CardFooter>
      </form>
    </Card>
  );
}
