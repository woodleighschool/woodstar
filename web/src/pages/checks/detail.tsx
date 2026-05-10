import { Link, useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { CheckStatusBadge } from "@/components/checks/check-status-badge";
import {
  DetailSettings,
  EditButton,
  PlatformBadge,
  SettingItem,
  ShowQueryButton,
  TargetSummary,
} from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useCheck, useCheckHosts } from "@/hooks/use-checks";
import { formatRelative } from "@/lib/utils";

export function CheckDetailPage() {
  const { checkId } = useParams({ from: "/_authenticated/checks/$checkId" });
  const check = useCheck(checkId);
  const hosts = useCheckHosts(checkId);

  if (check.error) {
    return (
      <div className="p-6">
        <Alert variant="destructive">
          <AlertTitle>Failed to load check</AlertTitle>
          <AlertDescription>{check.error.message}</AlertDescription>
        </Alert>
      </div>
    );
  }
  if (!check.data) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading check...
      </div>
    );
  }

  const hostRows = hosts.data?.items ?? [];
  const failing = hostRows.filter((row) => row.passes === false).length;
  const passing = hostRows.filter((row) => row.passes === true).length;

  return (
    <div className="flex flex-col gap-5 p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-2xl font-semibold tracking-tight">{check.data.name}</h1>
          {check.data.description ? (
            <p className="text-muted-foreground mt-1 max-w-3xl text-sm">{check.data.description}</p>
          ) : null}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <ShowQueryButton sql={check.data.query} />
          <EditButton to="/checks/$checkId/edit" params={{ checkId }}>
            Edit check
          </EditButton>
        </div>
      </div>

      <DetailSettings>
        <SettingItem label="Hosts failing">{failing}</SettingItem>
        <SettingItem label="Hosts passing">{passing}</SettingItem>
        <SettingItem label="Targeted platforms">
          <PlatformBadge platform={check.data.platform} />
        </SettingItem>
        <SettingItem label="Targets">
          <TargetSummary scope={check.data.label_scope} platform={check.data.platform} />
        </SettingItem>
      </DetailSettings>

      {check.data.resolution ? (
        <div className="rounded-md border bg-muted/20 p-4">
          <h2 className="text-sm font-semibold">Resolution</h2>
          <p className="text-muted-foreground mt-1 text-sm">{check.data.resolution}</p>
        </div>
      ) : null}

      <div className="grid gap-3">
        <div>
          <h2 className="text-base font-semibold">Hosts</h2>
          <p className="text-muted-foreground text-sm">Latest check result for each host.</p>
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
                      <CheckStatusBadge passes={row.passes} />
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {row.first_failed_at
                        ? formatRelative(row.first_failed_at)
                        : row.last_evaluated_at
                          ? formatRelative(row.last_evaluated_at)
                          : "-"}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
}
