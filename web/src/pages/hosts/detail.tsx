import { Link, useParams } from "@tanstack/react-router";
import { ChevronLeft, ServerCog } from "lucide-react";

import { ErrorState } from "@/components/feedback/error-state";
import { PendingBanner } from "@/components/feedback/pending-banner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { Spinner } from "@/components/ui/spinner";
import { StatusDot } from "@/components/ui/status-dot";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useHost } from "@/hooks/use-host";
import { endpoints } from "@/lib/endpoints";
import { formatBytes, formatRelative } from "@/lib/utils";

function DetailRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[10rem_1fr] gap-3 px-4 py-2 border-b last:border-b-0 text-sm">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-medium break-all">{value ?? "—"}</dd>
    </div>
  );
}

export function HostDetailPage() {
  const { hostId } = useParams({ from: "/_authed/hosts/$hostId" });
  const { data: host, query, isPending } = useHost(hostId);

  const headerStatus = host?.status === "online" ? "online" : "offline";
  const title = host?.display_name ?? `Host ${hostId}`;

  return (
    <div className="flex flex-col">
      <PageHeader
        title={title}
        description={
          host
            ? `${host.hardware_model} • ${host.hardware_serial}`
            : "Loading host…"
        }
        actions={
          <Button asChild variant="outline" size="sm">
            <Link to="/hosts" className="gap-1">
              <ChevronLeft className="size-4" /> Back to hosts
            </Link>
          </Button>
        }
      />

      <div className="flex items-center gap-2 px-6 py-3 border-b bg-muted/30 text-xs">
        <StatusDot status={headerStatus} />
        <span className="font-medium uppercase tracking-wide">{headerStatus}</span>
        {host ? (
          <span className="text-muted-foreground">
            • last seen {formatRelative(host.last_seen_at)}
          </span>
        ) : null}
      </div>

      <div className="p-6 space-y-4">
        {isPending ? (
          <PendingBanner endpoint={endpoints.host(hostId).path} />
        ) : null}

        {query.isError ? (
          <ErrorState
            message={query.error?.message}
            onRetry={() => query.refetch()}
          />
        ) : null}

        {query.isLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Spinner /> Loading host detail…
          </div>
        ) : null}

        <Tabs defaultValue="details">
          <TabsList>
            <TabsTrigger value="details">Details</TabsTrigger>
            <TabsTrigger value="software">Software</TabsTrigger>
            <TabsTrigger value="munki">Munki Issues</TabsTrigger>
            <TabsTrigger value="reports">Reports</TabsTrigger>
            <TabsTrigger value="checks">Checks</TabsTrigger>
          </TabsList>

          <TabsContent value="details">
            {host ? (
              <div className="rounded-lg border bg-card">
                <dl>
                  <DetailRow label="Display name" value={host.display_name} />
                  <DetailRow label="Hostname" value={host.hostname} />
                  <DetailRow label="Computer name" value={host.computer_name} />
                  <DetailRow label="Serial" value={host.hardware_serial} />
                  <DetailRow label="UUID" value={host.uuid} />
                  <DetailRow label="OS version" value={host.os_version} />
                  <DetailRow label="osquery" value={host.osquery_version} />
                  <DetailRow label="Model" value={host.hardware_model} />
                  <DetailRow label="CPU" value={host.cpu_type} />
                  <DetailRow label="Memory" value={formatBytes(host.memory_bytes)} />
                  <DetailRow
                    label="Disk"
                    value={`${formatBytes(host.disk_available_bytes)} free of ${formatBytes(host.disk_total_bytes)}`}
                  />
                  <DetailRow label="Primary IP" value={host.primary_ip} />
                  <DetailRow label="Public IP" value={host.public_ip} />
                  <DetailRow label="MAC" value={host.primary_mac} />
                  <DetailRow
                    label="FileVault"
                    value={
                      host.filevault_enabled === undefined
                        ? "—"
                        : host.filevault_enabled
                          ? "Enabled"
                          : "Disabled"
                    }
                  />
                  <DetailRow label="MDM server" value={host.mdm_server_url} />
                  <DetailRow label="Munki" value={host.munki_version} />
                  <DetailRow
                    label="Labels"
                    value={
                      host.labels.length === 0 ? (
                        "—"
                      ) : (
                        <div className="flex flex-wrap gap-1">
                          {host.labels.map((label) => (
                            <Badge key={label.id} variant="outline">
                              {label.name}
                            </Badge>
                          ))}
                        </div>
                      )
                    }
                  />
                </dl>
              </div>
            ) : !isPending && !query.isLoading ? (
              <EmptyState
                icon={ServerCog}
                title="Host not found"
                description="This host does not exist, or hasn't been enrolled yet."
              />
            ) : null}
          </TabsContent>

          <TabsContent value="software">
            <PendingBanner endpoint={endpoints.hostSoftware(hostId).path} />
          </TabsContent>
          <TabsContent value="munki">
            <PendingBanner endpoint={endpoints.hostMunkiIssues(hostId).path} />
          </TabsContent>
          <TabsContent value="reports">
            <PendingBanner endpoint={endpoints.hostReports(hostId).path} />
          </TabsContent>
          <TabsContent value="checks">
            <PendingBanner endpoint={endpoints.hostChecks(hostId).path} />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
