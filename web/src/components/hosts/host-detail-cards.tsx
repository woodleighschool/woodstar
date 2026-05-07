import type { ReactNode } from "react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { Host } from "@/hooks/use-hosts";
import { deviceMappingSourceLabel } from "@/lib/device-mapping-source-labels";
import { formatBytes, formatRelative } from "@/lib/utils";

interface Tile {
  label: string;
  value: ReactNode;
}

export function HostInfoCard({ host }: { host: Host }) {
  const tiles: Tile[] = [];

  tiles.push({
    label: "Agent",
    value: host.osquery_version || host.orbit_version ? agentSummary(host.osquery_version, host.orbit_version) : "-",
  });

  const battery = host.batteries?.[0];
  if (battery?.health) {
    tiles.push({ label: "Battery condition", value: battery.health });
  }

  if (host.disk_space_available_bytes != null) {
    tiles.push({
      label: "Disk space available",
      value: `${formatBytes(host.disk_space_available_bytes)}${diskPercent(host)}`,
    });
  }

  tiles.push({
    label: "Enrolled",
    value: host.enrolled_at ? (
      <span title={new Date(host.enrolled_at).toLocaleString()}>{formatRelative(host.enrolled_at)}</span>
    ) : (
      "-"
    ),
  });

  tiles.push({ label: "Hardware model", value: host.hardware_model || "-" });

  if (host.last_restarted_at) {
    tiles.push({
      label: "Last restarted",
      value: (
        <span title={new Date(host.last_restarted_at).toLocaleString()}>{formatRelative(host.last_restarted_at)}</span>
      ),
    });
  }

  tiles.push({
    label: "MAC address",
    value: host.primary_mac ? <span className="font-mono text-xs">{host.primary_mac}</span> : "-",
  });

  if (host.physical_memory > 0) {
    tiles.push({ label: "Memory", value: formatBytes(host.physical_memory) });
  }

  tiles.push({ label: "Operating system", value: host.os_version || "-" });

  tiles.push({ label: "Private IP address", value: host.primary_ip || "-" });

  if (host.cpu_brand || host.cpu_type) {
    tiles.push({ label: "Processor type", value: host.cpu_brand || host.cpu_type });
  }

  tiles.push({ label: "Public IP address", value: host.public_ip || "-" });

  tiles.push({ label: "Serial number", value: host.hardware_serial || "-" });

  tiles.sort((a, b) => a.label.localeCompare(b.label));

  return (
    <Card>
      <CardContent className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5">
        {tiles.map((t) => (
          <div key={t.label} className="flex min-w-0 flex-col gap-1">
            <dt className="text-muted-foreground text-xs font-semibold">{t.label}</dt>
            <dd className="text-foreground truncate text-sm">{t.value}</dd>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

export function HostLabelsCard({ host }: { host: Host }) {
  const labels = host.labels ?? [];
  if (labels.length === 0) return null;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Labels</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex flex-wrap gap-1.5">
          {labels.map((l) => (
            <Badge key={l.id} variant="outline" className="font-normal">
              {l.name}
            </Badge>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

export function HostUsersCard({ host }: { host: Host }) {
  const primary = host.device_mappings?.[0];
  const users = (host.users ?? []).filter((u) => u.username);
  if (users.length === 0 && !primary) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Local users</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {primary ? (
          <p className="text-sm">
            <span className="text-muted-foreground">Primary user: </span>
            {primary.email}{" "}
            <span className="text-muted-foreground text-xs">({deviceMappingSourceLabel(primary.source)})</span>
          </p>
        ) : null}
        {users.length > 0 ? (
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Username</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Directory</TableHead>
                  <TableHead>Shell</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.map((u) => (
                  <TableRow key={u.uid || u.username}>
                    <TableCell className="font-mono text-xs">{u.username}</TableCell>
                    <TableCell className="text-muted-foreground">{u.type || "-"}</TableCell>
                    <TableCell className="text-muted-foreground font-mono text-xs">{u.directory || "-"}</TableCell>
                    <TableCell className="text-muted-foreground font-mono text-xs">{u.shell || "-"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function agentSummary(osqueryVersion: string, orbitVersion: string): ReactNode {
  if (osqueryVersion && orbitVersion) {
    return (
      <span>
        osquery {osqueryVersion} <span className="text-muted-foreground">·</span> Orbit {orbitVersion}
      </span>
    );
  }
  return osqueryVersion || orbitVersion;
}

function diskPercent(host: Host) {
  const available = host.disk_space_available_bytes;
  const total = host.disk_space_total_bytes;
  if (available == null || total == null || total <= 0) return "";
  return ` (${((available / total) * 100).toFixed(0)}%)`;
}
