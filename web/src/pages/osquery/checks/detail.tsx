import { Link, useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { CheckStatusBadge } from "@/components/osquery/checks/check-status-badge";
import {
  DetailSettings,
  EditButton,
  LiveRunButton,
  SettingItem,
  ShowQueryButton,
  TargetSummary,
} from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useCheck, useCheckHosts } from "@/hooks/use-checks";
import { formatRelative } from "@/lib/utils";

export function CheckDetailPage() {
  const { checkId } = useParams({ from: "/_authenticated/osquery/checks/$checkId" });
  const checkID = Number(checkId);
  const check = useCheck(checkID);
  const hosts = useCheckHosts(checkID);

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

  const hostRows = hosts.data?.items ?? [];
  const failing = hostRows.filter((row) => row.response === "fail").length;
  const passing = hostRows.filter((row) => row.response === "pass").length;

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
        <SettingItem label="Hosts Failing">{failing}</SettingItem>
        <SettingItem label="Hosts Passing">{passing}</SettingItem>
        <SettingItem label="Targets">
          <TargetSummary scope={check.data.label_scope} />
        </SettingItem>
      </DetailSettings>

      <div className="grid gap-3">
        <div>
          <h2 className="text-base font-semibold">Hosts</h2>
        </div>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Host</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Since</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {hostRows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="text-muted-foreground h-28 text-center">
                    This check has not evaluated any hosts yet.
                  </TableCell>
                </TableRow>
              ) : (
                hostRows.map((row) => (
                  <TableRow key={`${row.check_id}-${row.host_id}`}>
                    <TableCell>
                      <Link
                        to="/hosts/$hostId"
                        params={{ hostId: String(row.host_id) }}
                        className="font-medium hover:underline"
                      >
                        {row.host_name || String(row.host_id)}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <CheckStatusBadge response={row.response} />
                    </TableCell>
                    <TableCell>{row.updated_at ? formatRelative(row.updated_at) : "-"}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </PageShell>
  );
}
