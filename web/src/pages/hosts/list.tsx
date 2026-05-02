import { Filter, KeyRound, Package, ServerCog } from "lucide-react";

import { OrbitEnrollSecretsDialog } from "@/components/credentials/orbit-enroll-secrets-dialog";
import { ErrorState } from "@/components/feedback/error-state";
import { PendingBanner } from "@/components/feedback/pending-banner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/data-table";
import { EmptyState } from "@/components/ui/empty-state";
import { Input } from "@/components/ui/input";
import { PageHeader } from "@/components/ui/page-header";
import { Spinner } from "@/components/ui/spinner";
import { StatusDot } from "@/components/ui/status-dot";
import { useHosts } from "@/hooks/use-hosts";
import { endpoints } from "@/lib/endpoints";
import { formatBytes, formatRelative } from "@/lib/utils";

export function HostsListPage() {
  const { data, query, isPending } = useHosts();

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Hosts"
        description="Orbit/osquery-managed Macs in this Woodstar deployment."
        actions={
          <>
            <OrbitEnrollSecretsDialog
              trigger={
                <Button variant="outline" size="sm" className="gap-2">
                  <KeyRound className="size-4" /> Manage enroll secrets
                </Button>
              }
            />
            <Button size="sm" className="gap-2" disabled>
              <Package className="size-4" /> Generate enrollment package
            </Button>
          </>
        }
      />

      <div className="flex flex-col gap-4 p-6">
        <div className="flex flex-wrap items-center gap-2">
          <Input
            type="search"
            placeholder="Search hostname, serial, UUID…"
            className="max-w-sm"
            disabled={isPending}
          />
          <Button variant="outline" size="sm" className="gap-2" disabled={isPending}>
            <Filter className="size-4" /> Filters
          </Button>
          <div className="ml-auto text-xs text-muted-foreground">
            {isPending
              ? "endpoint pending"
              : query.isFetching
                ? "loading…"
                : `${data.length} hosts`}
          </div>
        </div>

        {isPending ? <PendingBanner endpoint={endpoints.hosts.path} /> : null}

        {query.isError ? (
          <ErrorState
            message={query.error?.message}
            onRetry={() => query.refetch()}
          />
        ) : null}

        {query.isLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Spinner /> Loading hosts…
          </div>
        ) : null}

        {!isPending && !query.isLoading && !query.isError && data.length === 0 ? (
          <EmptyState
            icon={ServerCog}
            title="No hosts enrolled"
            description="Once a Mac runs Orbit with a valid enroll secret it will appear here."
          />
        ) : null}

        {data.length > 0 ? (
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-8" aria-label="Status" />
                  <TableHead>Display name</TableHead>
                  <TableHead>Serial</TableHead>
                  <TableHead>OS version</TableHead>
                  <TableHead>Model</TableHead>
                  <TableHead>Memory</TableHead>
                  <TableHead>Last seen</TableHead>
                  <TableHead className="text-right">Issues</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((host) => (
                  <TableRow key={host.id}>
                    <TableCell>
                      <StatusDot
                        status={host.status === "online" ? "online" : "offline"}
                      />
                    </TableCell>
                    <TableCell className="font-medium">{host.display_name}</TableCell>
                    <TableCell className="font-mono text-xs">
                      {host.hardware_serial}
                    </TableCell>
                    <TableCell>{host.os_version}</TableCell>
                    <TableCell>{host.hardware_model}</TableCell>
                    <TableCell>{formatBytes(host.memory_bytes ?? 0)}</TableCell>
                    <TableCell>{formatRelative(host.last_seen_at)}</TableCell>
                    <TableCell className="text-right">
                      {host.issues_count > 0 ? (
                        <Badge variant="warning">{host.issues_count}</Badge>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : null}
      </div>
    </div>
  );
}
