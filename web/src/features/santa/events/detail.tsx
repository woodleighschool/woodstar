import { useParams, useSearch } from "@tanstack/react-router";
import { Check, X } from "lucide-react";

import { TableSurface } from "@components/data-table/table-surface";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@components/layout/scrollable-tabs";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { TabsContent, TabsTrigger } from "@components/ui/tabs";
import type { SantaExecutionEvent } from "@lib/api";
import { formatDateTime, isRecord } from "@lib/utils";

import { ExecutionDecisionBadge, HostLink, Timestamp } from "./event-ui";
import { useSantaEvent } from "./queries";

export function SantaEventDetailPage() {
  const { id: eventId } = useParams({
    from: "/_authenticated/santa/events/$id",
  });
  const search = useSearch({ from: "/_authenticated/santa/events/$id" });
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
  const entitlements = entitlementEntries(
    isRecord(executable.entitlements) ? executable.entitlements : {},
  );
  const signingChain = executable.signing_chain ?? [];
  const hasSigningChain = signingChain.length > 0;
  const hasEntitlements = entitlements.length > 0;
  const hasTabs = hasSigningChain || hasEntitlements;
  const activeView = executionDetailView(search.view, {
    hasSigningChain,
    hasEntitlements,
  });
  const hasBundle = Boolean(
    executable.file_bundle_hash ||
    executable.file_bundle_id ||
    executable.file_bundle_name ||
    executable.file_bundle_path,
  );
  const detailsContent = (
    <div className="flex flex-col gap-5">
      <ExecutionSection event={event} />
      <BinarySection event={event} />
      {hasBundle ? <BundleSection event={event} /> : null}
      <SessionsSection event={event} />
    </div>
  );

  return (
    <PageShell className="gap-6">
      <PageHeader title="Execution" description={event.file_path} />

      {hasTabs ? (
        <ScrollableTabs value={activeView}>
          <ScrollableTabsList>
            <TabsTrigger
              value="details"
              render={
                <Link
                  to="/santa/events/$id"
                  params={{ id: eventId }}
                  search={(previous) => ({ ...previous, view: undefined })}
                />
              }
              nativeButton={false}
            >
              Details
            </TabsTrigger>
            {hasSigningChain ? (
              <TabsTrigger
                value="signing-chain"
                render={
                  <Link
                    to="/santa/events/$id"
                    params={{ id: eventId }}
                    search={(previous) => ({
                      ...previous,
                      view: "signing-chain",
                    })}
                  />
                }
                nativeButton={false}
              >
                Signing Chain
              </TabsTrigger>
            ) : null}
            {hasEntitlements ? (
              <TabsTrigger
                value="entitlements"
                render={
                  <Link
                    to="/santa/events/$id"
                    params={{ id: eventId }}
                    search={(previous) => ({
                      ...previous,
                      view: "entitlements",
                    })}
                  />
                }
                nativeButton={false}
              >
                Entitlements
              </TabsTrigger>
            ) : null}
          </ScrollableTabsList>

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
        </ScrollableTabs>
      ) : (
        detailsContent
      )}
    </PageShell>
  );
}

function ExecutionSection({ event }: { event: SantaExecutionEvent }) {
  return (
    <KeyValueSection title="Execution">
      <KeyValueRow label="Decision" value={<ExecutionDecisionBadge decision={event.decision} />} />
      <KeyValueRow label="Host" value={<HostLink host={event.host} />} />
      <KeyValueRow label="Executing User" value={event.executing_user} />
      <KeyValueRow label="PID" value={formatNumber(event.pid)} />
      <KeyValueRow label="Parent PID" value={formatNumber(event.ppid)} />
      <KeyValueRow label="Parent Process" value={event.parent_name} />
      <KeyValueRow label="Occurred" value={<Timestamp value={event.occurred_at} />} />
      <KeyValueRow label="Ingested" value={formatDateTime(event.ingested_at)} />
    </KeyValueSection>
  );
}

function BinarySection({ event }: { event: SantaExecutionEvent }) {
  const executable = event.executable;

  return (
    <KeyValueSection title="Binary">
      <KeyValueRow label="File Name" value={executable.file_name} />
      <KeyValueRow label="Path" value={event.file_path} />
      <KeyValueRow label="SHA-256" value={executable.sha256} />
      <KeyValueRow label="CDHash" value={executable.cdhash} />
      <KeyValueRow label="Signing ID" value={executable.signing_id} />
      <KeyValueRow label="Team ID" value={executable.team_id} />
      <KeyValueRow label="Signing Status" value={formatEnumValue(executable.signing_status)} />
      <KeyValueRow label="CS Flags" value={formatCodeSigningFlags(executable.codesigning_flags)} />
      <KeyValueRow
        label="Secure Signing Time"
        value={formatDateTime(executable.secure_signing_time)}
      />
      <KeyValueRow label="Signing Time" value={formatDateTime(executable.signing_time)} />
    </KeyValueSection>
  );
}

function BundleSection({ event }: { event: SantaExecutionEvent }) {
  const executable = event.executable;

  return (
    <KeyValueSection title="Bundle">
      <KeyValueRow label="Bundle ID" value={executable.file_bundle_id} />
      <KeyValueRow label="Name" value={executable.file_bundle_name} />
      <KeyValueRow label="Path" value={executable.file_bundle_path} />
      <KeyValueRow label="Executable Rel Path" value={executable.file_bundle_executable_rel_path} />
      <KeyValueRow label="Version" value={executable.file_bundle_version} />
      <KeyValueRow label="Version String" value={executable.file_bundle_version_string} />
      <KeyValueRow label="Bundle Hash" value={executable.file_bundle_hash} />
      <KeyValueRow label="Binary Count" value={formatNumber(executable.file_bundle_binary_count)} />
      <KeyValueRow label="Hash Time" value={formatMillis(executable.file_bundle_hash_millis)} />
    </KeyValueSection>
  );
}

function SessionsSection({ event }: { event: SantaExecutionEvent }) {
  return (
    <KeyValueSection title="Sessions">
      <KeyValueRow
        label="Logged-In Users"
        value={<ValueList values={event.logged_in_users ?? []} />}
      />
      <KeyValueRow
        label="Current Sessions"
        value={<ValueList values={event.current_sessions ?? []} />}
      />
    </KeyValueSection>
  );
}

function SigningChainTable({
  signingChain,
}: {
  signingChain: NonNullable<SantaExecutionEvent["executable"]["signing_chain"]>;
}) {
  return (
    <TableSurface>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Certificate</TableHead>
            <TableHead>Organization</TableHead>
            <TableHead>Organizational Unit</TableHead>
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
                {cert.organization ?? "-"}
              </TableCell>
              <TableCell className="min-w-40 whitespace-normal">
                {cert.organizational_unit ?? "-"}
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
    </TableSurface>
  );
}

function EntitlementsTable({ entitlements }: { entitlements: EntitlementEntry[] }) {
  return (
    <TableSurface>
      <Table>
        <TableHeader>
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
    </TableSurface>
  );
}

function ValueText({ value }: { value?: string }) {
  if (!value) return <span className="text-muted-foreground">-</span>;
  return (
    <span className="wrap-break-word" title={value}>
      {value}
    </span>
  );
}

function ValueList({ values }: { values: string[] }) {
  const cleaned = values.filter(Boolean);
  if (cleaned.length === 0) return <span className="text-sm text-muted-foreground">-</span>;
  return <span className="wrap-break-word">{cleaned.join(", ")}</span>;
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
    return <ValueList values={normalized.map(formatEntitlementValue)} />;
  }

  if (typeof normalized === "string" || typeof normalized === "number") {
    return <ValueText value={String(normalized)} />;
  }

  if (isRecord(normalized)) {
    return (
      <ValueList
        values={Object.entries(normalized).map(
          ([key, entryValue]) => `${key}: ${formatEntitlementValue(entryValue)}`,
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
      if (!isRecord(entry) || typeof entry.key !== "string" || entry.key === "") return [];
      return [{ key: entry.key, value: entry.value }];
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

function formatEntitlementValue(value: unknown): string {
  const normalized = normalizeEntitlementValue(value);
  if (normalized === null || normalized === undefined) return "";
  if (Array.isArray(normalized))
    return normalized.map(formatEntitlementValue).filter(Boolean).join(", ");
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

function executionDetailView(
  value: unknown,
  availability: { hasSigningChain: boolean; hasEntitlements: boolean },
) {
  if (value === "signing-chain" && availability.hasSigningChain) return value;
  if (value === "entitlements" && availability.hasEntitlements) return value;
  return "details";
}
