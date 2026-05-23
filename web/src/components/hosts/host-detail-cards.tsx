import { Link } from "@tanstack/react-router";
import type { ColumnDef, PaginationState, SortingState } from "@tanstack/react-table";
import { useMemo, useState, type ReactNode } from "react";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { Host, HostDetail } from "@/hooks/use-hosts";
import { deviceMappingSourceLabel } from "@/lib/device-mapping-source-labels";
import { formatBytes, formatRelative } from "@/lib/utils";

interface Tile {
  label: string;
  value: ReactNode;
}

type HostCertificate = NonNullable<HostDetail["certificates"]>[number];

const CERTIFICATES_PAGE_SIZE = 25;

export function HostInfoCard({ host }: { host: HostDetail }) {
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
    value: host.primary_mac || "-",
  });

  if (host.physical_memory > 0) {
    tiles.push({ label: "Memory", value: formatBytes(host.physical_memory) });
  }

  tiles.push({ label: "Operating system", value: host.os_version || "-" });

  tiles.push({ label: "Private IP address", value: host.primary_ip ?? "-" });

  if (host.cpu_brand || host.cpu_type) {
    tiles.push({ label: "Processor type", value: host.cpu_brand || host.cpu_type });
  }

  tiles.push({ label: "Public IP address", value: host.public_ip ?? "-" });

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

export function HostLabelsCard({ host }: { host: HostDetail }) {
  const labels = (host.labels ?? []).filter((l) => l.label_type === "regular");
  if (labels.length === 0) return null;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Labels</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex flex-wrap gap-1.5">
          {labels.map((l) => (
            <Button key={l.id} asChild size="xs" variant="outline" className="font-normal">
              <Link to="/hosts" search={{ label_id: String(l.id) }}>
                {l.name}
              </Link>
            </Button>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

export function HostUsersCard({ host }: { host: HostDetail }) {
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
                    <TableCell>{u.username}</TableCell>
                    <TableCell className="text-muted-foreground">{u.type || "-"}</TableCell>
                    <TableCell className="text-muted-foreground">{u.directory || "-"}</TableCell>
                    <TableCell className="text-muted-foreground">{u.shell || "-"}</TableCell>
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

export function HostCertificatesCard({ host }: { host: HostDetail }) {
  const [selectedCertificate, setSelectedCertificate] = useState<HostCertificate | null>(null);
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: CERTIFICATES_PAGE_SIZE });
  const [sorting, setSorting] = useState<SortingState>([{ id: "common_name", desc: false }]);
  const certificates = useMemo(() => host.certificates ?? [], [host.certificates]);
  if (certificates.length === 0) return null;

  return (
    <Card className="gap-4 py-4">
      <CardHeader className="flex flex-row items-center justify-between gap-3">
        <CardTitle>Certificates</CardTitle>
        <span className="text-muted-foreground text-xs tabular-nums">
          {certificates.length} {certificates.length === 1 ? "certificate" : "certificates"}
        </span>
      </CardHeader>
      <CardContent>
        <DataTable
          columns={certificateColumns}
          data={certificates}
          totalCount={certificates.length}
          pagination={pagination}
          sorting={sorting}
          onPaginationChange={setPagination}
          onSortingChange={setSorting}
          onRowClick={setSelectedCertificate}
          getRowId={(certificate) => String(certificate.id)}
          empty={<span className="text-muted-foreground text-sm">No certificates</span>}
        />
        <CertificateDetailsDialog certificate={selectedCertificate} onOpenChange={setSelectedCertificate} />
      </CardContent>
    </Card>
  );
}

const certificateColumns: ColumnDef<HostCertificate>[] = [
  {
    id: "common_name",
    accessorFn: (certificate) => certificateName(certificate),
    header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
    cell: ({ row }) => (
      <span className="block truncate font-medium" title={certificateName(row.original)}>
        {certificateName(row.original)}
      </span>
    ),
    meta: {
      cellClassName: "max-w-[360px] py-1.5",
      headClassName: "h-8",
    },
  },
  {
    id: "issuer",
    accessorFn: (certificate) => certificate.issuer.common_name,
    header: ({ column }) => <DataTableColumnHeader column={column} title="Issuer" />,
    cell: ({ row }) => (
      <span className="text-muted-foreground block max-w-[360px] truncate" title={row.original.issuer.common_name}>
        {row.original.issuer.common_name || "-"}
      </span>
    ),
    meta: {
      cellClassName: "py-1.5",
      headClassName: "h-8",
    },
  },
  {
    id: "source",
    accessorFn: (certificate) => certificateKeychain(certificate),
    header: ({ column }) => <DataTableColumnHeader column={column} title="Keychain" />,
    cell: ({ row }) => (
      <span className="text-muted-foreground" title={row.original.username !== "" ? row.original.username : undefined}>
        {certificateKeychain(row.original)}
      </span>
    ),
    meta: {
      cellClassName: "py-1.5",
      headClassName: "h-8",
    },
  },
  {
    id: "not_valid_before",
    accessorKey: "not_valid_before",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Issued" />,
    cell: ({ row }) =>
      row.original.not_valid_before ? (
        <span className="text-muted-foreground" title={new Date(row.original.not_valid_before).toLocaleString()}>
          {formatDate(row.original.not_valid_before)}
        </span>
      ) : (
        <span className="text-muted-foreground">-</span>
      ),
    meta: {
      cellClassName: "py-1.5",
      headClassName: "h-8",
    },
  },
  {
    id: "not_valid_after",
    accessorKey: "not_valid_after",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Expires" />,
    cell: ({ row }) =>
      row.original.not_valid_after ? (
        <span className="text-muted-foreground" title={new Date(row.original.not_valid_after).toLocaleString()}>
          {formatDate(row.original.not_valid_after)}
        </span>
      ) : (
        <span className="text-muted-foreground">-</span>
      ),
    meta: {
      cellClassName: "py-1.5",
      headClassName: "h-8",
    },
  },
];

function CertificateDetailsDialog({
  certificate,
  onOpenChange,
}: {
  certificate: HostCertificate | null;
  onOpenChange: (certificate: HostCertificate | null) => void;
}) {
  return (
    <Dialog open={certificate !== null} onOpenChange={(open) => !open && onOpenChange(null)}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Certificate details</DialogTitle>
          <DialogDescription>{certificate ? certificateName(certificate) : ""}</DialogDescription>
        </DialogHeader>
        {certificate ? (
          <div className="grid gap-5">
            <CertificateDetailSection
              title="Subject"
              rows={certificateNameRows(certificate.subject, certificate.common_name)}
            />
            <CertificateDetailSection title="Issuer" rows={certificateNameRows(certificate.issuer)} />
            <CertificateDetailSection
              title="Validity"
              rows={[
                ["Issued", formatDate(certificate.not_valid_before)],
                ["Expires", formatDate(certificate.not_valid_after)],
                ["Certificate authority", certificate.certificate_authority ? "Yes" : "No"],
              ]}
            />
            <CertificateDetailSection
              title="Key"
              rows={[
                ["Algorithm", certificate.key_algorithm],
                ["Strength", certificate.key_strength ? `${certificate.key_strength} bits` : ""],
                ["Usage", certificate.key_usage],
                ["Serial", certificate.serial],
              ]}
            />
            <CertificateDetailSection
              title="Signature and keychain"
              rows={[
                ["Signing algorithm", certificate.signing_algorithm],
                ["Keychain", certificateKeychain(certificate)],
                ["Username", certificate.username],
              ]}
            />
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function CertificateDetailSection({ title, rows }: { title: string; rows: Array<[string, ReactNode]> }) {
  const visibleRows = rows.filter(([, value]) => value !== "" && value !== null && value !== undefined);
  if (visibleRows.length === 0) return null;

  return (
    <section className="grid gap-2">
      <h3 className="text-sm font-medium">{title}</h3>
      <dl className="grid grid-cols-[140px_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm">
        {visibleRows.map(([label, value]) => (
          <div key={label} className="contents">
            <dt className="text-muted-foreground">{label}</dt>
            <dd className="min-w-0 break-words">{value}</dd>
          </div>
        ))}
      </dl>
    </section>
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

function certificateKeychain(certificate: HostCertificate) {
  if (certificate.source === "system") return "System";
  if (certificate.source === "user") return "User";
  return firstNonEmpty(certificate.source, "-");
}

function certificateName(certificate: HostCertificate) {
  return firstNonEmpty(
    certificate.common_name,
    certificate.subject.common_name,
    certificate.serial,
    `Certificate ${certificate.id}`,
  );
}

function certificateNameRows(name: HostCertificate["subject"], commonNameFallback = ""): Array<[string, ReactNode]> {
  return [
    ["Country", name.country],
    ["Organization", name.organization],
    ["Organizational unit", name.organizational_unit],
    ["Common name", firstNonEmpty(name.common_name, commonNameFallback)],
  ];
}

function firstNonEmpty(...values: string[]) {
  return values.find((value) => value !== "") ?? "";
}

function formatDate(value: string | null | undefined) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

function diskPercent(host: Host) {
  const available = host.disk_space_available_bytes;
  const total = host.disk_space_total_bytes;
  if (available == null || total == null || total <= 0) return "";
  return ` (${((available / total) * 100).toFixed(0)}%)`;
}
