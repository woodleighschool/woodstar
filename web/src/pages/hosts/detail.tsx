import { Link, useParams } from "@tanstack/react-router";
import { ChevronLeft } from "lucide-react";

import { ErrorState } from "@/components/feedback/error-state";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/data-table";
import { PageHeader } from "@/components/ui/page-header";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useHost, useHostSoftware, type Host } from "@/hooks/use-hosts";
import { deviceMappingSourceLabel } from "@/lib/device-mapping-source-labels";
import { softwareSourceLabel } from "@/lib/software-source-labels";
import { formatBytes, formatRelative } from "@/lib/utils";

export function HostDetailPage() {
  const { hostId } = useParams({ from: "/_authed/hosts/$hostId" });
  const query = useHost(hostId);
  const host = query.data;

  return (
    <div className="flex flex-col">
      <PageHeader
        title={host?.display_name || `Host ${hostId}`}
        description={host?.hardware_uuid}
        actions={
          <Button asChild variant="outline" size="sm">
            <Link to="/hosts" className="gap-1">
              <ChevronLeft className="size-4" /> Back to hosts
            </Link>
          </Button>
        }
      />

      <div className="p-6">
        {query.error ? (
          <ErrorState message={query.error.message} onRetry={() => query.refetch()} />
        ) : query.isLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Spinner /> Loading…
          </div>
        ) : !host ? null : (
          <Tabs defaultValue="details">
            <TabsList>
              <TabsTrigger value="details">Details</TabsTrigger>
              <TabsTrigger value="software">Software</TabsTrigger>
            </TabsList>

            <TabsContent value="details">
              <DetailsTab host={host} />
            </TabsContent>

            <TabsContent value="software">
              <SoftwareTab hostId={hostId} />
            </TabsContent>
          </Tabs>
        )}
      </div>
    </div>
  );
}

function DetailsTab({ host }: { host: Host }) {
  const primaryMapping = host.device_mappings?.[0];
  const primaryUser = primaryMapping
    ? `${primaryMapping.email} (${deviceMappingSourceLabel(primaryMapping.source)})`
    : "-";

  return (
    <DefinitionList
      rows={[
        ["Hardware UUID", <span className="font-mono text-xs">{host.hardware_uuid}</span>],
        ["Vendor", host.hardware_vendor || "-"],
        ["Model", host.hardware_model || "-"],
        ["CPU", host.cpu_brand ? `${host.cpu_brand} (${host.cpu_logical_cores} logical cores)` : "-"],
        ["Memory", host.physical_memory > 0 ? formatBytes(host.physical_memory) : "-"],
        ["Hostname", host.hostname || "-"],
        ["Computer name", host.computer_name || "-"],
        ["Platform", host.platform ? <Badge variant="muted">{host.platform}</Badge> : "-"],
        ["OS version", host.os_version || "-"],
        ["Kernel", host.kernel_version || "-"],
        ["Serial", host.hardware_serial || "-"],
        ["osquery version", host.osquery_version || "-"],
        ["Orbit version", host.orbit_version || "-"],
        ["Primary user", primaryUser],
        [
          "Enrolled",
          host.enrolled_at ? (
            <span title={new Date(host.enrolled_at).toLocaleString()}>{formatRelative(host.enrolled_at)}</span>
          ) : (
            "-"
          ),
        ],
        [
          "Last seen",
          host.last_seen_at ? (
            <span title={new Date(host.last_seen_at).toLocaleString()}>{formatRelative(host.last_seen_at)}</span>
          ) : (
            "-"
          ),
        ],
        [
          "Detail refreshed",
          host.detail_updated_at ? (
            <span title={new Date(host.detail_updated_at).toLocaleString()}>
              {formatRelative(host.detail_updated_at)}
            </span>
          ) : (
            <span className="text-muted-foreground">never</span>
          ),
        ],
      ]}
    />
  );
}

function SoftwareTab({ hostId }: { hostId: string }) {
  const query = useHostSoftware(hostId);

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
      <div className="rounded-md border border-dashed bg-muted/30 px-4 py-6 text-sm">
        <p className="font-medium">No software inventory yet</p>
        <p className="text-muted-foreground">osquery will populate this on next detail refresh.</p>
      </div>
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Version</TableHead>
            <TableHead>Source</TableHead>
            <TableHead>Last opened</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row) => (
            <TableRow key={row.id}>
              <TableCell className="font-medium">{row.name}</TableCell>
              <TableCell className="text-muted-foreground">{row.version || "-"}</TableCell>
              <TableCell className="text-muted-foreground" title={row.source}>
                {softwareSourceLabel(row.source)}
              </TableCell>
              <TableCell
                className="text-muted-foreground"
                title={row.last_opened_at ? new Date(row.last_opened_at).toLocaleString() : ""}
              >
                {row.last_opened_at ? formatRelative(row.last_opened_at) : "-"}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function DefinitionList({ rows }: { rows: Array<[string, React.ReactNode]> }) {
  return (
    <dl className="rounded-lg border bg-card divide-y">
      {rows.map(([label, value]) => (
        <div key={label} className="grid grid-cols-[10rem_1fr] gap-3 px-4 py-2 text-sm">
          <dt className="text-muted-foreground">{label}</dt>
          <dd className="font-medium break-all">{value ?? "-"}</dd>
        </div>
      ))}
    </dl>
  );
}
