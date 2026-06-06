import { Link, useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { useCheck } from "@/hooks/use-checks";
import { LiveRunner } from "@/pages/osquery/live-runner";

export function CheckLivePage() {
  const { checkId } = useParams({ from: "/_authenticated/osquery/checks/$checkId" });
  const check = useCheck(Number(checkId));

  if (check.error) {
    return (
      <PageShell>
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Check</AlertTitle>
          <AlertDescription>{check.error.message}</AlertDescription>
        </Alert>
      </PageShell>
    );
  }
  if (!check.data) {
    return (
      <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading Check...
      </PageShell>
    );
  }

  return (
    <LiveRunner
      kind="check"
      itemId={Number(checkId)}
      name={check.data.name}
      sql={check.data.query}
      editAction={
        <Button asChild variant="outline" size="sm">
          <Link to="/osquery/checks/$checkId" params={{ checkId }}>
            Check
          </Link>
        </Button>
      }
    />
  );
}
