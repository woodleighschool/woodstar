import { useForm } from "@tanstack/react-form";
import type { ColumnDef, PaginationState, SortingState } from "@tanstack/react-table";
import { Pencil, Trash2, UserPlus } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import { z } from "zod";

import { DataTable, DataTableColumnHeader } from "@/components/data-table";
import { EmptyPanel } from "@/components/empty-panel";
import { FormField } from "@/components/form-field";
import { manualUserAffinityMapping } from "@/components/hosts/host-user-affinity";
import { userAffinitySourceLabel } from "@/components/hosts/user-affinity-source-labels";
import { LabelChips } from "@/components/labels/label-chips";
import { SubmitButton } from "@/components/submit-button";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FieldError, FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuth } from "@/hooks/use-auth";
import { useClearHostUserAffinity, useSetHostUserAffinity, type Host, type HostDetail } from "@/hooks/use-hosts";
import { requiredString } from "@/lib/form-validation";
import { cn, formatBytes, formatRelative } from "@/lib/utils";

interface Tile {
  label: string;
  value: ReactNode;
}

type HostCertificate = NonNullable<HostDetail["certificates"]>[number];

const CERTIFICATES_PAGE_SIZE = 25;
const certificateSourceLabels: Record<string, string> = {
  system: "System",
  user: "User",
};

export function HostInfoCard({ host }: { host: HostDetail }) {
  const tiles: Tile[] = [];
  const osqueryVersion = host.agents.osquery.version;
  const orbitVersion = host.agents.orbit.version;

  tiles.push({
    label: "Agent",
    value:
      orbitVersion || osqueryVersion ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <span>{orbitVersion || osqueryVersion}</span>
          </TooltipTrigger>
          <TooltipContent>
            <div className="whitespace-pre-line">{`osquery: ${osqueryVersion || "-"}\nOrbit: ${orbitVersion || "-"}`}</div>
          </TooltipContent>
        </Tooltip>
      ) : (
        "-"
      ),
  });

  const battery = host.batteries?.[0];
  if (battery?.health) {
    tiles.push({ label: "Battery Condition", value: battery.health });
  }

  if (host.storage.boot_volume.available_bytes != null) {
    tiles.push({
      label: "Disk Space Available",
      value: `${formatBytes(host.storage.boot_volume.available_bytes)}${diskPercent(host)}`,
    });
  }

  tiles.push({
    label: "Enrolled",
    value: host.enrollment.enrolled_at ? (
      <span title={new Date(host.enrollment.enrolled_at).toLocaleString()}>
        {formatRelative(host.enrollment.enrolled_at)}
      </span>
    ) : (
      "-"
    ),
  });

  tiles.push({ label: "Hardware Model", value: host.hardware.model_identifier || "-" });

  if (host.timestamps.last_restarted_at) {
    tiles.push({
      label: "Last Restarted",
      value: (
        <span title={new Date(host.timestamps.last_restarted_at).toLocaleString()}>
          {formatRelative(host.timestamps.last_restarted_at)}
        </span>
      ),
    });
  }

  tiles.push({
    label: "MAC Address",
    value: host.network.primary_mac || "-",
  });

  if (host.hardware.memory_bytes > 0) {
    tiles.push({ label: "Memory", value: formatBytes(host.hardware.memory_bytes) });
  }

  tiles.push({ label: "Operating System", value: osDisplayName(host) });

  tiles.push({ label: "Private IP Address", value: host.network.primary_ip ?? "-" });

  if (host.hardware.cpu.brand || host.hardware.cpu.architecture) {
    tiles.push({ label: "Processor Type", value: host.hardware.cpu.brand || host.hardware.cpu.architecture });
  }

  tiles.push({ label: "Public IP Address", value: host.network.last_remote_ip ?? "-" });

  tiles.push({ label: "Serial Number", value: host.hardware.serial || "-" });

  tiles.sort((a, b) => a.label.localeCompare(b.label));

  return (
    <Card>
      <CardContent>
        <dl className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5">
          {tiles.map((t) => (
            <div key={t.label} className="flex min-w-0 flex-col gap-1">
              <dt className="text-muted-foreground text-xs font-semibold">{t.label}</dt>
              <dd className="text-foreground truncate text-sm">{t.value}</dd>
            </div>
          ))}
        </dl>
      </CardContent>
    </Card>
  );
}

export function HostIdentityCard({ host }: { host: HostDetail }) {
  const { user } = useAuth();
  const [dialogOpen, setDialogOpen] = useState(false);
  const affinity = host.user_affinity.primary;
  const canEdit = user?.role === "admin";
  const hasManualMapping = manualUserAffinityMapping(host.user_affinity.mappings) !== null;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-3">
        <CardTitle>User Affinity</CardTitle>
        {canEdit ? (
          <Button type="button" variant="outline" size="sm" onClick={() => setDialogOpen(true)}>
            {hasManualMapping ? <Pencil /> : <UserPlus />}
            {hasManualMapping ? "Edit user" : "Set user"}
          </Button>
        ) : null}
      </CardHeader>
      <CardContent>
        <dl className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5">
          <IdentityItem label="Name" value={displayValue(affinity?.name)} />
          <IdentityItem label="Username" value={displayValue(affinity?.username)} />
          <IdentityItem label="Email" value={displayValue(affinity?.email)} />
          <IdentityItem label="Department" value={displayValue(affinity?.department)} />
          <IdentityItem label="Source" value={affinity ? userAffinitySourceLabel(affinity.source) : "-"} />
          <IdentityItem
            label="Groups"
            value={<UserGroups groups={affinity?.groups ?? []} />}
            className="sm:col-span-2"
          />
        </dl>
      </CardContent>
      {dialogOpen ? <HostUserMappingDialog host={host} onOpenChange={setDialogOpen} /> : null}
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
        <LabelChips labels={labels} />
      </CardContent>
    </Card>
  );
}

export function HostUsersCard({ host }: { host: HostDetail }) {
  const users = (host.users ?? []).filter((u) => u.username);
  if (users.length === 0) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Local User Accounts</CardTitle>
      </CardHeader>
      <CardContent>
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
                  <TableCell className="font-medium">{u.username}</TableCell>
                  <TableCell>{u.type || "-"}</TableCell>
                  <TableCell>{u.directory || "-"}</TableCell>
                  <TableCell>{u.shell || "-"}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}

function IdentityItem({ label, value, className }: { label: string; value: ReactNode; className?: string }) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-1", className)}>
      <dt className="text-muted-foreground text-xs font-semibold">{label}</dt>
      <dd className="text-foreground break-words text-sm">{value}</dd>
    </div>
  );
}

function UserGroups({ groups }: { groups: readonly string[] }) {
  if (groups.length === 0) return <span>-</span>;
  return (
    <div className="max-h-24 overflow-y-auto pr-1">
      <div className="flex flex-wrap gap-1.5">
        {groups.map((group) => (
          <Badge key={group} variant="secondary" className="font-normal">
            {group}
          </Badge>
        ))}
      </div>
    </div>
  );
}

function displayValue(value: string | null | undefined) {
  return value && value.trim() !== "" ? value : "-";
}

function HostUserMappingDialog({ host, onOpenChange }: { host: HostDetail; onOpenChange: (open: boolean) => void }) {
  const manual = manualUserAffinityMapping(host.user_affinity.mappings);
  const setMapping = useSetHostUserAffinity();
  const clearMapping = useClearHostUserAffinity();
  const pending = setMapping.isPending || clearMapping.isPending;
  const submitError = setMapping.error ?? clearMapping.error;

  const form = useForm({
    defaultValues: { email: manual?.email ?? host.user_affinity.primary?.email ?? "" },
    validators: { onSubmit: z.object({ email: requiredString("Email / UPN") }) },
    onSubmit: async ({ value }) => {
      await setMapping.mutateAsync({ id: host.id, body: { email: value.email } });
      onOpenChange(false);
    },
  });

  async function handleClear() {
    await clearMapping.mutateAsync(host.id);
    onOpenChange(false);
  }

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{manual ? "Edit User Affinity" : "Set User Affinity"}</DialogTitle>
          <DialogDescription>Set the email or UPN Woodstar should prefer for this host.</DialogDescription>
        </DialogHeader>
        <form
          noValidate
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit();
          }}
        >
          <FieldGroup className="gap-4">
            <form.Field name="email">
              {(field) => (
                <FormField field={field} label="Email / UPN" htmlFor="host-user-email" required>
                  {(control) => (
                    <Input
                      {...control}
                      type="email"
                      autoComplete="off"
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  )}
                </FormField>
              )}
            </form.Field>
          </FieldGroup>

          {submitError ? <FieldError>{submitError.message}</FieldError> : null}

          <DialogFooter className="pt-2">
            {manual ? (
              <Button type="button" variant="ghost" size="sm" disabled={pending} onClick={() => void handleClear()}>
                <Trash2 />
                Clear
              </Button>
            ) : null}
            <Button type="button" variant="ghost" size="sm" disabled={pending} onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <SubmitButton pending={pending} size="sm">
              Save
            </SubmitButton>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
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
      <CardHeader>
        <CardTitle>Certificates</CardTitle>
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
          empty={<EmptyPanel>No certificates yet</EmptyPanel>}
        />
        <CertificateDetailsDialog certificate={selectedCertificate} onOpenChange={setSelectedCertificate} />
      </CardContent>
    </Card>
  );
}

const certificateColumns: ColumnDef<HostCertificate>[] = [
  {
    id: "common_name",
    accessorKey: "common_name",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
    cell: ({ row }) => row.original.common_name || "-",
    meta: {
      cellClassName: "max-w-[360px] py-1.5",
      headClassName: "h-8",
    },
  },
  {
    id: "issuer",
    accessorFn: (certificate) => certificate.issuer.common_name,
    header: ({ column }) => <DataTableColumnHeader column={column} title="Issuer" />,
    cell: ({ row }) => row.original.issuer.common_name || "-",
    meta: {
      cellClassName: "py-1.5",
      headClassName: "h-8",
    },
  },
  {
    id: "source",
    accessorFn: (certificate) => certificateSourceLabel(certificate.source),
    header: ({ column }) => <DataTableColumnHeader column={column} title="Keychain" />,
    cell: ({ row }) => certificateSourceLabel(row.original.source),
    meta: {
      cellClassName: "py-1.5",
      headClassName: "h-8",
    },
  },
  {
    id: "not_valid_before",
    accessorKey: "not_valid_before",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Issued" />,
    cell: ({ row }) => (row.original.not_valid_before ? formatDate(row.original.not_valid_before) : "-"),
    meta: {
      cellClassName: "py-1.5",
      headClassName: "h-8",
    },
  },
  {
    id: "not_valid_after",
    accessorKey: "not_valid_after",
    header: ({ column }) => <DataTableColumnHeader column={column} title="Expires" />,
    cell: ({ row }) => (row.original.not_valid_after ? formatDate(row.original.not_valid_after) : "-"),
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
          <DialogTitle>Certificate Details</DialogTitle>
          <DialogDescription>{certificate?.common_name ?? ""}</DialogDescription>
        </DialogHeader>
        {certificate ? (
          <div className="grid gap-5">
            <CertificateDetailSection title="Subject" rows={certificateNameRows(certificate.subject)} />
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
              title="Signature and Keychain"
              rows={[
                ["Signing algorithm", certificate.signing_algorithm],
                ["Keychain", certificateSourceLabel(certificate.source)],
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

function certificateSourceLabel(source: string) {
  return certificateSourceLabels[source] ?? source;
}

function certificateNameRows(name: HostCertificate["subject"]): Array<[string, ReactNode]> {
  return [
    ["Country", name.country],
    ["Organization", name.organization],
    ["Organizational unit", name.organizational_unit],
    ["Common name", name.common_name],
  ];
}

function formatDate(value: string | null | undefined) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

function diskPercent(host: Host) {
  const available = host.storage.boot_volume.available_bytes;
  const total = host.storage.boot_volume.total_bytes;
  if (available == null || total == null || total <= 0) return "";
  return ` (${((available / total) * 100).toFixed(0)}%)`;
}

function osDisplayName(host: Host) {
  const parts = [host.os.name, host.os.version].filter(Boolean);
  if (parts.length > 0) return parts.join(" ");
  return host.os.build || "-";
}
