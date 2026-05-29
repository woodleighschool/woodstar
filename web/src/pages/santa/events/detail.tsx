import { useParams } from "@tanstack/react-router";
import { FileCode2, Loader2 } from "lucide-react";
import type { ReactNode } from "react";

import { PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useSantaEvent, type SantaEvent } from "@/hooks/use-santa";
import { cn } from "@/lib/utils";

import { executableLabel, fileName } from "./constants";
import { ExecutionDecisionBadge, HostLink, Timestamp } from "./event-ui";

interface Tile {
  label: string;
  value: ReactNode;
}

export function SantaEventDetailPage() {
  const { eventId } = useParams({ from: "/_authenticated/santa/events/$eventId" });
  const id = Number(eventId);
  const query = useSantaEvent(Number.isFinite(id) ? id : null);

  if (query.error) {
    return (
      <PageShell>
        <Alert variant="destructive">
          <AlertTitle>Failed to Load Event</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      </PageShell>
    );
  }
  if (query.isLoading || !query.data) {
    return (
      <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading...
      </PageShell>
    );
  }

  const event = query.data;
  const executable = event.executable;
  const entitlements = entitlementEntries(executable.entitlements ?? {});
  const signingChain = executable.signing_chain ?? [];
  const hasBundle = Boolean(
    executable.file_bundle_hash ||
    executable.file_bundle_id ||
    executable.file_bundle_name ||
    executable.file_bundle_path,
  );

  return (
    <PageShell className="gap-6">
      <EventHeader event={event} />
      <ExecutionCard event={event} />
      <div className="grid gap-5 xl:grid-cols-2">
        <BinaryCard event={event} />
        {hasBundle ? <BundleCard event={event} /> : null}
        <SessionsCard event={event} />
      </div>
      <SigningChainCard signingChain={signingChain} />
      <EntitlementsCard entitlements={entitlements} />
    </PageShell>
  );
}

function EventHeader({ event }: { event: SantaEvent }) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div className="flex min-w-0 items-center gap-4">
        <div className="bg-muted/40 flex size-12 shrink-0 items-center justify-center rounded-md border">
          <FileCode2 className="text-muted-foreground size-6" />
        </div>
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <h1 className="text-foreground truncate text-xl font-semibold" title={executableLabel(event)}>
              {executableLabel(event)}
            </h1>
            <ExecutionDecisionBadge decision={event.decision} />
          </div>
          <p className="text-muted-foreground truncate font-mono text-xs">
            {event.file_path || event.executable.sha256}
          </p>
        </div>
      </div>
    </div>
  );
}

function ExecutionCard({ event }: { event: SantaEvent }) {
  const tiles: Tile[] = [
    { label: "Host", value: <HostLink host={event.host} /> },
    { label: "Executing User", value: <ValueText value={event.executing_user} /> },
    { label: "PID", value: <ValueText value={formatNumber(event.pid)} /> },
    { label: "Parent PID", value: <ValueText value={formatNumber(event.ppid)} /> },
    { label: "Parent Process", value: <ValueText value={event.parent_name} /> },
    {
      label: "Occurred",
      value: <Timestamp value={event.occurred_at} />,
    },
    { label: "Ingested", value: <ValueText value={formatDate(event.ingested_at)} /> },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Execution</CardTitle>
      </CardHeader>
      <CardContent>
        <TileGrid tiles={tiles} />
      </CardContent>
    </Card>
  );
}

function BinaryCard({ event }: { event: SantaEvent }) {
  const executable = event.executable;
  const tiles: Tile[] = [
    { label: "File Name", value: <ValueText value={executable.file_name || fileName(event.file_path)} /> },
    { label: "Path", value: <CodeText value={event.file_path} /> },
    { label: "SHA-256", value: <CodeText value={executable.sha256} breakAll /> },
    { label: "CDHash", value: <CodeText value={executable.cdhash} breakAll /> },
    { label: "Signing ID", value: <CodeText value={executable.signing_id} breakAll /> },
    { label: "Team ID", value: <CodeText value={executable.team_id} /> },
    { label: "Signing Status", value: <ValueText value={formatEnumValue(executable.signing_status)} /> },
    { label: "CS Flags", value: <ValueText value={formatCodeSigningFlags(executable.codesigning_flags)} /> },
    { label: "Secure Signing Time", value: <ValueText value={formatDate(executable.secure_signing_time)} /> },
    { label: "Signing Time", value: <ValueText value={formatDate(executable.signing_time)} /> },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Binary</CardTitle>
      </CardHeader>
      <CardContent>
        <TileGrid tiles={tiles} />
      </CardContent>
    </Card>
  );
}

function BundleCard({ event }: { event: SantaEvent }) {
  const executable = event.executable;
  const tiles: Tile[] = [
    { label: "Bundle ID", value: <CodeText value={executable.file_bundle_id} /> },
    { label: "Name", value: <ValueText value={executable.file_bundle_name} /> },
    { label: "Path", value: <CodeText value={executable.file_bundle_path} /> },
    { label: "Executable Rel Path", value: <CodeText value={executable.file_bundle_executable_rel_path} /> },
    { label: "Version", value: <ValueText value={executable.file_bundle_version} /> },
    { label: "Version String", value: <ValueText value={executable.file_bundle_version_string} /> },
    { label: "Bundle Hash", value: <CodeText value={executable.file_bundle_hash} breakAll /> },
    { label: "Binary Count", value: <ValueText value={formatNumber(executable.file_bundle_binary_count)} /> },
    { label: "Hash Time", value: <ValueText value={formatMillis(executable.file_bundle_hash_millis)} /> },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Bundle</CardTitle>
      </CardHeader>
      <CardContent>
        <TileGrid tiles={tiles} />
      </CardContent>
    </Card>
  );
}

function SessionsCard({ event }: { event: SantaEvent }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Sessions</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4 md:grid-cols-2">
        <SessionGroup label="Logged-In Users" values={event.logged_in_users ?? []} />
        <SessionGroup label="Current Sessions" values={event.current_sessions ?? []} />
      </CardContent>
    </Card>
  );
}

function SigningChainCard({ signingChain }: { signingChain: NonNullable<SantaEvent["executable"]["signing_chain"]> }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Signing Chain</CardTitle>
      </CardHeader>
      <CardContent>
        {signingChain.length === 0 ? (
          <EmptyBlock>No signing chain.</EmptyBlock>
        ) : (
          <div className="overflow-hidden rounded-lg border">
            <Table>
              <TableHeader className="bg-muted">
                <TableRow>
                  <TableHead>Position</TableHead>
                  <TableHead>Certificate</TableHead>
                  <TableHead>Organization</TableHead>
                  <TableHead>SHA-256</TableHead>
                  <TableHead>Valid From</TableHead>
                  <TableHead>Valid Until</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {signingChain.map((cert, index) => (
                  <TableRow key={`${cert.sha256}:${cert.common_name ?? ""}:${cert.valid_from ?? ""}`}>
                    <TableCell>{index === 0 ? "Leaf" : `CA ${index}`}</TableCell>
                    <TableCell className="min-w-64 whitespace-normal">{cert.common_name ?? "-"}</TableCell>
                    <TableCell className="min-w-40 whitespace-normal">
                      {cert.organization ?? cert.organizational_unit ?? "-"}
                    </TableCell>
                    <TableCell className="min-w-64 whitespace-normal">
                      <CodeText value={cert.sha256} breakAll />
                    </TableCell>
                    <TableCell>{formatDate(cert.valid_from)}</TableCell>
                    <TableCell>{formatDate(cert.valid_until)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function EntitlementsCard({ entitlements }: { entitlements: EntitlementEntry[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Entitlements</CardTitle>
      </CardHeader>
      <CardContent>
        {entitlements.length === 0 ? (
          <EmptyBlock>No entitlements.</EmptyBlock>
        ) : (
          <div className="overflow-hidden rounded-lg border">
            <Table>
              <TableHeader className="bg-muted">
                <TableRow>
                  <TableHead>Entitlement</TableHead>
                  <TableHead>Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {entitlements.map((entitlement) => (
                  <TableRow key={entitlement.key}>
                    <TableCell className="min-w-72 whitespace-normal">
                      <CodeText value={entitlement.key} />
                    </TableCell>
                    <TableCell className="whitespace-normal">
                      <CodeText value={entitlement.value} breakAll />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function TileGrid({ tiles }: { tiles: Tile[] }) {
  return (
    <dl className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5">
      {tiles.map((tile) => (
        <div key={tile.label} className="flex min-w-0 flex-col gap-1">
          <dt className="text-muted-foreground text-xs font-semibold">{tile.label}</dt>
          <dd className="text-foreground min-w-0 text-sm">{tile.value}</dd>
        </div>
      ))}
    </dl>
  );
}

function SessionGroup({ label, values }: { label: string; values: string[] }) {
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="text-muted-foreground text-xs font-semibold">{label}</div>
      <ValueBadges values={values} />
    </div>
  );
}

function CodeText({ value, breakAll = false }: { value?: string; breakAll?: boolean }) {
  if (!value) return <span className="text-muted-foreground">-</span>;
  return (
    <span className={cn("font-mono text-xs", breakAll ? "break-all" : "block truncate")} title={value}>
      {value}
    </span>
  );
}

function ValueText({ value }: { value?: string }) {
  if (!value) return <span className="text-muted-foreground">-</span>;
  return value;
}

function ValueBadges({ values }: { values: string[] }) {
  const cleaned = values.filter(Boolean);
  if (cleaned.length === 0) return <span className="text-muted-foreground text-sm">-</span>;
  return (
    <div className="flex flex-wrap gap-1.5">
      {cleaned.map((value) => (
        <Badge key={value} variant="outline" className="font-mono text-xs font-normal">
          {value}
        </Badge>
      ))}
    </div>
  );
}

function EmptyBlock({ children }: { children: ReactNode }) {
  return <div className="bg-muted/30 rounded-md border border-dashed px-4 py-6 text-sm">{children}</div>;
}

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : "-";
}

function formatNumber(value?: number) {
  return value ? String(value) : "";
}

function formatMillis(value?: number) {
  return value ? `${value} ms` : "";
}

function formatCodeSigningFlags(value?: number) {
  if (!value) return "";
  return `0x${value.toString(16).toUpperCase()} (${value})`;
}

function formatEnumValue(value?: string) {
  if (!value || value === "unspecified") return "";
  return value
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

type EntitlementEntry = {
  key: string;
  value: string;
};

function entitlementEntries(raw: Record<string, unknown>): EntitlementEntry[] {
  const santaEntries = raw.entitlements;
  if (Array.isArray(santaEntries)) {
    const entries = santaEntries.flatMap((entry) => {
      if (!entry || typeof entry !== "object") return [];
      const record = entry as Record<string, unknown>;
      if (typeof record.key !== "string" || record.key === "") return [];
      return [{ key: record.key, value: formatEntitlementValue(record.value) }];
    });
    if (entries.length > 0) return entries;
  }

  return Object.entries(raw)
    .filter(([key]) => key !== "entitlementsFiltered")
    .map(([key, value]) => ({ key, value: formatEntitlementValue(value) }));
}

function formatEntitlementValue(value: unknown) {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value);
}
