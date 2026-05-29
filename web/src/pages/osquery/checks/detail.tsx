import { useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import {
  DetailSettings,
  EditButton,
  LiveRunButton,
  SettingItem,
  ShowQueryButton,
  TargetSummary,
} from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { useCheck } from "@/hooks/use-checks";

export function CheckDetailPage() {
  const { checkId } = useParams({ from: "/_authenticated/osquery/checks/$checkId" });
  const checkID = Number(checkId);
  const check = useCheck(checkID);

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
    <PageShell>
      <PageHeader
        title={check.data.name}
        description={check.data.description}
        actions={
          <>
            <ShowQueryButton sql={check.data.query} />
            <LiveRunButton to="/osquery/checks/$checkId/live" params={{ checkId }} />
            <EditButton to="/osquery/checks/$checkId/edit" params={{ checkId }}>
              Edit Check
            </EditButton>
          </>
        }
      />

      <DetailSettings>
        <SettingItem label="Hosts Failing">{check.data.failing_host_count}</SettingItem>
        <SettingItem label="Hosts Passing">{check.data.passing_host_count}</SettingItem>
        <SettingItem label="Targets">
          <TargetSummary scope={check.data.label_scope} />
        </SettingItem>
      </DetailSettings>
    </PageShell>
  );
}
