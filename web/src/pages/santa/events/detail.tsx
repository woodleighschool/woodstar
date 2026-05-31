import { useParams } from "@tanstack/react-router";
import { Check, FileCode2, Loader2, X } from "lucide-react";
import type { ReactNode } from "react";

import { PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useSantaEvent, type SantaEvent } from "@/hooks/use-santa";

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
  const hasSigningChain = signingChain.length > 0;
  const hasEntitlements = entitlements.length > 0;
  const hasTabs = hasSigningChain || hasEntitlements;
  const hasBundle = Boolean(
    executable.file_bundle_hash ||
    executable.file_bundle_id ||
    executable.file_bundle_name ||
    executable.file_bundle_path,
  );
  const detailsContent = (
    <div className="flex flex-col gap-5">
      <ExecutionCard event={event} />
      <div className="grid gap-5 xl:grid-cols-2">
        <BinaryCard event={event} />
        {hasBundle ? <BundleCard event={event} /> : null}
        <SessionsCard event={event} />
      </div>
    </div>
  );

  return (
    <PageShell className="gap-6">
      <EventHeader event={event} />

      {hasTabs ? (
        <Tabs defaultValue="details">
          <TabsList>
            <TabsTrigger value="details">Details</TabsTrigger>
            {hasSigningChain ? <TabsTrigger value="signing-chain">Signing Chain</TabsTrigger> : null}
            {hasEntitlements ? <TabsTrigger value="entitlements">Entitlements</TabsTrigger> : null}
          </TabsList>

          <TabsContent value="details">{detailsContent}</TabsContent>

          {hasSigningChain ? (
            <TabsContent value="signing-chain">
              <SigningChainTable signingChain={signingChain} />
            </TabsContent>
          ) : null}

          {hasEntitlements ? (
            <TabsContent value="entitlements">
              <EntitlementsTable entitlements={entitlements} />
            </TabsContent>
          ) : null}
        </Tabs>
      ) : (
        detailsContent
      )}
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
          <p className="text-muted-foreground truncate text-sm">{event.file_path || event.executable.sha256}</p>
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
    { label: "Path", value: <ValueText value={event.file_path} /> },
    { label: "SHA-256", value: <ValueText value={executable.sha256} /> },
    { label: "CDHash", value: <ValueText value={executable.cdhash} /> },
    { label: "Signing ID", value: <ValueText value={executable.signing_id} /> },
    { label: "Team ID", value: <ValueText value={executable.team_id} /> },
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
    { label: "Bundle ID", value: <ValueText value={executable.file_bundle_id} /> },
    { label: "Name", value: <ValueText value={executable.file_bundle_name} /> },
    { label: "Path", value: <ValueText value={executable.file_bundle_path} /> },
    { label: "Executable Rel Path", value: <ValueText value={executable.file_bundle_executable_rel_path} /> },
    { label: "Version", value: <ValueText value={executable.file_bundle_version} /> },
    { label: "Version String", value: <ValueText value={executable.file_bundle_version_string} /> },
    { label: "Bundle Hash", value: <ValueText value={executable.file_bundle_hash} /> },
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

function SigningChainTable({ signingChain }: { signingChain: NonNullable<SantaEvent["executable"]["signing_chain"]> }) {
  return (
    <div className="overflow-hidden rounded-lg border">
      <Table>
        <TableHeader className="bg-muted">
          <TableRow>
            <TableHead>Certificate</TableHead>
            <TableHead>Organization</TableHead>
            <TableHead>SHA-256</TableHead>
            <TableHead>Valid From</TableHead>
            <TableHead>Valid Until</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {signingChain.map((cert) => (
            <TableRow key={`${cert.sha256}:${cert.common_name ?? ""}:${cert.valid_from ?? ""}`}>
              <TableCell className="min-w-64 whitespace-normal">{cert.common_name ?? "-"}</TableCell>
              <TableCell className="min-w-40 whitespace-normal">
                {cert.organization ?? cert.organizational_unit ?? "-"}
              </TableCell>
              <TableCell className="min-w-64 whitespace-normal">
                <ValueText value={cert.sha256} />
              </TableCell>
              <TableCell>{formatDate(cert.valid_from)}</TableCell>
              <TableCell>{formatDate(cert.valid_until)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function EntitlementsTable({ entitlements }: { entitlements: EntitlementEntry[] }) {
  return (
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
                <ValueText value={entitlement.key} />
              </TableCell>
              <TableCell className="whitespace-normal">
                <EntitlementValue value={entitlement.value} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
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

function ValueText({ value }: { value?: string }) {
  if (!value) return <span className="text-muted-foreground">-</span>;
  return (
    <span className="break-words" title={value}>
      {value}
    </span>
  );
}

function ValueBadges({ values }: { values: string[] }) {
  const cleaned = values.filter(Boolean);
  if (cleaned.length === 0) return <span className="text-muted-foreground text-sm">-</span>;
  return (
    <div className="flex flex-wrap gap-1.5">
      {cleaned.map((value) => (
        <Badge key={value} variant="outline" className="font-normal">
          {value}
        </Badge>
      ))}
    </div>
  );
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
  value: unknown;
};

function EntitlementValue({ value }: { value: unknown }) {
  const normalized = normalizeEntitlementValue(value);

  if (normalized === null || normalized === undefined || normalized === "") {
    return <span className="text-muted-foreground">-</span>;
  }

  if (typeof normalized === "boolean") {
    const Icon = normalized ? Check : X;
    return <Icon className="size-4" />;
  }

  if (Array.isArray(normalized)) {
    return <ValueBadges values={normalized.map(formatEntitlementChip)} />;
  }

  if (typeof normalized === "string" || typeof normalized === "number") {
    return <ValueText value={String(normalized)} />;
  }

  if (isRecord(normalized)) {
    return (
      <ValueBadges
        values={Object.entries(normalized).map(([key, entryValue]) => `${key}: ${formatEntitlementChip(entryValue)}`)}
      />
    );
  }

  return <ValueText value={JSON.stringify(normalized)} />;
}

function entitlementEntries(raw: Record<string, unknown>): EntitlementEntry[] {
  const santaEntries = raw.entitlements;
  if (Array.isArray(santaEntries)) {
    const entries = santaEntries.flatMap((entry) => {
      if (!entry || typeof entry !== "object") return [];
      const record = entry as Record<string, unknown>;
      if (typeof record.key !== "string" || record.key === "") return [];
      return [{ key: record.key, value: record.value }];
    });
    if (entries.length > 0) return entries;
  }

  return Object.entries(raw)
    .filter(([key]) => key !== "entitlementsFiltered")
    .map(([key, value]) => ({ key, value }));
}

function normalizeEntitlementValue(value: unknown): unknown {
  if (typeof value !== "string") return value;

  const trimmed = value.trim();
  const looksEncoded =
    trimmed.startsWith("[") ||
    trimmed.startsWith("{") ||
    trimmed.startsWith('"') ||
    trimmed === "true" ||
    trimmed === "false" ||
    trimmed === "null" ||
    /^-?\d+(\.\d+)?$/.test(trimmed);

  if (!looksEncoded) return value;

  try {
    return JSON.parse(trimmed);
  } catch {
    return value;
  }
}

function formatEntitlementChip(value: unknown): string {
  const normalized = normalizeEntitlementValue(value);
  if (normalized === null || normalized === undefined) return "";
  if (Array.isArray(normalized)) return normalized.map(formatEntitlementChip).filter(Boolean).join(", ");
  if (typeof normalized === "string" || typeof normalized === "number" || typeof normalized === "boolean") {
    return String(normalized);
  }
  if (isRecord(normalized) && "identifier" in normalized) {
    const identifier = normalized.identifier;
    if (typeof identifier === "string") return identifier;
  }
  return JSON.stringify(normalized);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
