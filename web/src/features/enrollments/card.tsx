import { KeyRound } from "lucide-react";
import { useMemo, useState } from "react";

import { Button } from "@components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@components/ui/card";
import { Skeleton } from "@components/ui/skeleton";
import { useAuth } from "@features/auth/queries";
import type { Integration } from "@features/enrollments/metadata";
import { enrollmentCardDescription, enrollmentCardTitle } from "@features/enrollments/metadata";
import { useAgentSecrets } from "@features/enrollments/queries";
import { AgentSecretsDialog } from "@features/enrollments/secrets-dialog";

export function EnrollmentOverviewCard({ integration }: { integration: Integration }) {
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const query = useAgentSecrets(isAdmin);
  const count = useMemo(
    () => (query.data ?? []).filter((secret) => secret.agent === integration).length,
    [integration, query.data],
  );
  const [open, setOpen] = useState(false);

  if (!isAdmin) return null;

  return (
    <>
      <Card size="sm" className="min-w-0">
        <CardHeader>
          <CardTitle>{enrollmentCardTitle(integration)}</CardTitle>
          <CardDescription>{enrollmentCardDescription(integration)}</CardDescription>
          <CardAction>
            <Button size="sm" onClick={() => setOpen(true)}>
              <KeyRound data-icon="inline-start" />
              Manage
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          {query.isLoading ? (
            <Skeleton className="h-9 w-24" />
          ) : query.error ? (
            <p className="text-sm text-destructive">Secrets unavailable</p>
          ) : (
            <span className="text-3xl font-semibold tracking-tight tabular-nums">
              {count.toLocaleString()}
            </span>
          )}
        </CardContent>
      </Card>

      {open ? <AgentSecretsDialog integration={integration} open onOpenChange={setOpen} /> : null}
    </>
  );
}
