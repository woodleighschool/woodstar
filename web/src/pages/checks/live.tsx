import { Link, useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { useCheck } from "@/hooks/use-checks";
import { LiveRunner } from "@/pages/live-runner";

export function CheckLivePage() {
  const { checkId } = useParams({ from: "/_authenticated/checks/$checkId" });
  const check = useCheck(checkId);

  if (check.error) {
    return (
      <PageShell>
        <Alert variant="destructive">
          <AlertTitle>Failed to load check</AlertTitle>
          <AlertDescription>{check.error.message}</AlertDescription>
        </Alert>
      </PageShell>
    );
  }
  if (!check.data) {
    return (
      <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading check...
      </PageShell>
    );
  }

  return (
    <LiveRunner
      kind="check"
      itemId={checkId}
      name={check.data.name}
      sql={check.data.query}
      editAction={
        <Button asChild variant="outline" size="sm">
          <Link to="/checks/$checkId/edit" params={{ checkId }}>
            Edit check
          </Link>
        </Button>
      }
    />
  );
}
