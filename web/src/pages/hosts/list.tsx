import { Link } from "@tanstack/react-router";
import { KeyRound, ServerCog } from "lucide-react";

import { ErrorState } from "@/components/feedback/error-state";
import { OrbitEnrollSecretsDialog } from "@/components/secrets/orbit-enroll-secrets-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Spinner } from "@/components/ui/spinner";
import { useHosts, type Host } from "@/hooks/use-hosts";
import { formatRelative } from "@/lib/utils";

export function HostsListPage() {
  const query = useHosts();

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Hosts"
        description="Orbit/osquery-managed Macs in this Woodstar deployment."
        actions={
          <OrbitEnrollSecretsDialog
            trigger={
              <Button variant="outline" size="sm" className="gap-2">
                <KeyRound className="size-4" /> Manage enroll secrets
              </Button>
            }
          />
        }
      />

      <div className="p-6">
        <HostsTable query={query} />
      </div>
    </div>
  );
}

function HostsTable({ query }: { query: ReturnType<typeof useHosts> }) {
  if (query.error) {
    return <ErrorState message={query.error.message} onRetry={() => query.refetch()} />;
  }

  if (query.isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Spinner /> Loading…
      </div>
    );
  }

  const data = query.data ?? [];
  if (data.length === 0) {
    return (
      <EmptyState
        icon={ServerCog}
        title="No hosts enrolled yet"
        description="Create an enroll secret, then point an Orbit-managed Mac at this Woodstar deployment. Hosts appear here on first check-in."
      />
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Primary user</TableHead>
            <TableHead>Platform</TableHead>
            <TableHead>Serial</TableHead>
            <TableHead>OS</TableHead>
            <TableHead>Last seen</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row) => {
            const primaryEmail = row.device_mappings?.[0]?.email ?? "";
            return (
              <TableRow key={row.id}>
                <TableCell className="font-medium">
                  <Link to="/hosts/$hostId" params={{ hostId: row.id }} className="hover:underline">
                    {row.display_name || row.hardware_uuid}
                  </Link>
                </TableCell>
                <TableCell className="text-muted-foreground max-w-[16rem] truncate" title={primaryEmail || ""}>
                  {primaryEmail || "-"}
                </TableCell>
                <TableCell>
                  <PlatformBadge platform={row.platform} />
                </TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">{row.hardware_serial || "-"}</TableCell>
                <TableCell className="text-muted-foreground">{row.os_version || "-"}</TableCell>
                <TableCell
                  className="text-muted-foreground"
                  title={row.last_seen_at ? new Date(row.last_seen_at).toLocaleString() : ""}
                >
                  {formatRelative(row.last_seen_at)}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

function PlatformBadge({ platform }: { platform: Host["platform"] }) {
  if (!platform) return <span className="text-muted-foreground">-</span>;
  return <Badge variant="muted">{platform}</Badge>;
}
