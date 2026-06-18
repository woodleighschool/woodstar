import { useParams } from "@tanstack/react-router";
import { Check, FileCode2, X } from "lucide-react";

import { KeyValueGrid, KeyValueItem } from "@/components/key-value";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryGate } from "@/components/query-gate";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { type SantaEvent, useSantaEvent } from "@/hooks/use-santa-events";
import { formatDateTime } from "@/lib/utils";

import { executableLabel, fileName } from "./decisions";
import { ExecutionDecisionBadge, HostLink, Timestamp } from "./event-ui";

export function SantaEventDetailPage() {
  const { eventId } = useParams({ from: "/_authenticated/santa/events/$eventId" });
  const id = Number(eventId);
  const query = useSantaEvent(Number.isFinite(id) ? id : null);

  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to load event"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
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
      <PageHeader
        leading={
          <div className="flex size-12 items-center justify-center rounded-md border bg-muted/40">
            <FileCode2 className="size-6 text-muted-foreground" />
          </div>
        }
        title={executableLabel(event)}
        description={event.file_path || event.executable.sha256}
        context={<ExecutionDecisionBadge decision={event.decision} />}
      />

      {hasTabs ? (
        <Tabs defaultValue="details">
          <TabsList>
            <TabsTrigger value="details">Details</TabsTrigger>
            {hasSigningChain ? (
              <TabsTrigger value="signing-chain">Signing Chain</TabsTrigger>
            ) : null}
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

function ExecutionCard({ event }: { event: SantaEvent }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Execution</CardTitle>
      </CardHeader>
      <CardContent>
        <KeyValueGrid>
          <KeyValueItem label="Host" value={<HostLink host={event.host} />} />
          <KeyValueItem label="Executing User" value={event.executing_user} />
          <KeyValueItem label="PID" value={formatNumber(event.pid)} />
          <KeyValueItem label="Parent PID" value={formatNumber(event.ppid)} />
          <KeyValueItem label="Parent Process" value={event.parent_name} />
          <KeyValueItem label="Occurred" value={<Timestamp value={event.occurred_at} />} />
          <KeyValueItem label="Ingested" value={formatDateTime(event.ingested_at)} />
        </KeyValueGrid>
      </CardContent>
    </Card>
  );
}

function BinaryCard({ event }: { event: SantaEvent }) {
  const executable = event.executable;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Binary</CardTitle>
      </CardHeader>
      <CardContent>
        <KeyValueGrid>
          <KeyValueItem
            label="File Name"
            value={executable.file_name || fileName(event.file_path)}
          />
          <KeyValueItem label="Path" value={event.file_path} />
          <KeyValueItem label="SHA-256" value={executable.sha256} />
          <KeyValueItem label="CDHash" value={executable.cdhash} />
          <KeyValueItem label="Signing ID" value={executable.signing_id} />
          <KeyValueItem label="Team ID" value={executable.team_id} />
          <KeyValueItem label="Signing Status" value={formatEnumValue(executable.signing_status)} />
          <KeyValueItem
            label="CS Flags"
            value={formatCodeSigningFlags(executable.codesigning_flags)}
          />
          <KeyValueItem
            label="Secure Signing Time"
            value={formatDateTime(executable.secure_signing_time)}
          />
          <KeyValueItem label="Signing Time" value={formatDateTime(executable.signing_time)} />
        </KeyValueGrid>
      </CardContent>
    </Card>
  );
}

function BundleCard({ event }: { event: SantaEvent }) {
  const executable = event.executable;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Bundle</CardTitle>
      </CardHeader>
      <CardContent>
        <KeyValueGrid>
          <KeyValueItem label="Bundle ID" value={executable.file_bundle_id} />
          <KeyValueItem label="Name" value={executable.file_bundle_name} />
          <KeyValueItem label="Path" value={executable.file_bundle_path} />
          <KeyValueItem
            label="Executable Rel Path"
            value={executable.file_bundle_executable_rel_path}
          />
          <KeyValueItem label="Version" value={executable.file_bundle_version} />
          <KeyValueItem label="Version String" value={executable.file_bundle_version_string} />
          <KeyValueItem label="Bundle Hash" value={executable.file_bundle_hash} />
          <KeyValueItem
            label="Binary Count"
            value={formatNumber(executable.file_bundle_binary_count)}
          />
          <KeyValueItem
            label="Hash Time"
            value={formatMillis(executable.file_bundle_hash_millis)}
          />
        </KeyValueGrid>
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

function SigningChainTable({
  signingChain,
}: {
  signingChain: NonNullable<SantaEvent["executable"]["signing_chain"]>;
}) {
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
              <TableCell className="min-w-64 whitespace-normal">
                {cert.common_name ?? "-"}
              </TableCell>
              <TableCell className="min-w-40 whitespace-normal">
                {cert.organization ?? cert.organizational_unit ?? "-"}
              </TableCell>
              <TableCell className="min-w-64 whitespace-normal">
                <ValueText value={cert.sha256} />
              </TableCell>
              <TableCell>{formatDateTime(cert.valid_from)}</TableCell>
              <TableCell>{formatDateTime(cert.valid_until)}</TableCell>
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

function SessionGroup({ label, values }: { label: string; values: string[] }) {
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="text-xs font-semibold text-muted-foreground">{label}</div>
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
  if (cleaned.length === 0) return <span className="text-sm text-muted-foreground">-</span>;
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
        values={Object.entries(normalized).map(
          ([key, entryValue]) => `${key}: ${formatEntitlementChip(entryValue)}`,
        )}
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
  if (Array.isArray(normalized))
    return normalized.map(formatEntitlementChip).filter(Boolean).join(", ");
  if (
    typeof normalized === "string" ||
    typeof normalized === "number" ||
    typeof normalized === "boolean"
  ) {
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
