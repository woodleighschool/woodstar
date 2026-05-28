import { useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { DetailSettings, SettingItem } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useSantaEvent } from "@/hooks/use-santa";

import { executableLabel, fileName } from "./constants";
import { ExecutionDecisionBadge, HostLink, Timestamp } from "./event-ui";

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
  const entitlements = Object.entries(event.executable.entitlements ?? {});
  const signingChain = event.executable.signing_chain ?? [];

  return (
    <PageShell className="gap-6">
      <PageHeader title={executableLabel(event)} description={event.file_path || event.executable.sha256} />

      <DetailSettings>
        <SettingItem label="Decision">
          <ExecutionDecisionBadge decision={event.decision} />
        </SettingItem>
        <SettingItem label="Host">
          <HostLink host={event.host} />
        </SettingItem>
        <SettingItem label="User">{event.executing_user || "-"}</SettingItem>
        <SettingItem label="Occurred">
          <Timestamp value={event.occurred_at} />
        </SettingItem>
        <SettingItem label="Ingested">{formatDate(event.ingested_at)}</SettingItem>
      </DetailSettings>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="signing">Signing</TabsTrigger>
          <TabsTrigger value="sessions">Sessions</TabsTrigger>
          <TabsTrigger value="entitlements">Entitlements</TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <div className="rounded-md border">
            <Table>
              <TableBody>
                <DetailRow label="Path" value={event.file_path} />
                <DetailRow label="File Name" value={event.executable.file_name || fileName(event.file_path)} />
                <DetailRow label="SHA-256" value={event.executable.sha256} breakAll />
                <DetailRow label="CDHash" value={event.executable.cdhash} breakAll />
                <DetailRow label="Bundle ID" value={event.executable.file_bundle_id} />
                <DetailRow label="Bundle Path" value={event.executable.file_bundle_path} />
                <DetailRow label="Signing ID" value={event.executable.signing_id} />
                <DetailRow label="Team ID" value={event.executable.team_id} />
              </TableBody>
            </Table>
          </div>
        </TabsContent>

        <TabsContent value="signing">
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Certificate</TableHead>
                  <TableHead>Organization</TableHead>
                  <TableHead>SHA-256</TableHead>
                  <TableHead>Valid From</TableHead>
                  <TableHead>Valid Until</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {signingChain.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="h-24 text-center">
                      No signing chain was reported.
                    </TableCell>
                  </TableRow>
                ) : (
                  signingChain.map((cert, index) => (
                    <TableRow key={`${cert.sha256}:${cert.common_name ?? ""}:${cert.valid_from ?? ""}`}>
                      <TableCell>{cert.common_name ?? (index === 0 ? "Leaf" : `Certificate ${index + 1}`)}</TableCell>
                      <TableCell>{cert.organization ?? cert.organizational_unit ?? "-"}</TableCell>
                      <TableCell className="break-all">{cert.sha256}</TableCell>
                      <TableCell>{formatDate(cert.valid_from)}</TableCell>
                      <TableCell>{formatDate(cert.valid_until)}</TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </TabsContent>

        <TabsContent value="sessions">
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Session</TableHead>
                  <TableHead>Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <SessionRows label="Logged-In User" values={event.logged_in_users ?? []} />
                <SessionRows label="Current Session" values={event.current_sessions ?? []} />
              </TableBody>
            </Table>
          </div>
        </TabsContent>

        <TabsContent value="entitlements">
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Entitlement</TableHead>
                  <TableHead>Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {entitlements.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={2} className="h-24 text-center">
                      No entitlements were reported.
                    </TableCell>
                  </TableRow>
                ) : (
                  entitlements.map(([key, value]) => (
                    <TableRow key={key}>
                      <TableCell>{key}</TableCell>
                      <TableCell className="break-all">{JSON.stringify(value)}</TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </TabsContent>
      </Tabs>
    </PageShell>
  );
}

function DetailRow({ label, value, breakAll = false }: { label: string; value?: string; breakAll?: boolean }) {
  return (
    <TableRow>
      <TableCell className="w-48">{label}</TableCell>
      <TableCell className={breakAll ? "break-all" : undefined}>{value ?? "-"}</TableCell>
    </TableRow>
  );
}

function SessionRows({ label, values }: { label: string; values: string[] }) {
  if (values.length === 0) {
    return (
      <TableRow>
        <TableCell>{label}</TableCell>
        <TableCell>-</TableCell>
      </TableRow>
    );
  }

  return values.map((value) => (
    <TableRow key={`${label}-${value}`}>
      <TableCell>{label}</TableCell>
      <TableCell>
        <Badge variant="outline">{value}</Badge>
      </TableCell>
    </TableRow>
  ));
}

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : "-";
}
