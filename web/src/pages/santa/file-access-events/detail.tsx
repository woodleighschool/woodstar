import { useParams } from "@tanstack/react-router";

import { EmptyPanel } from "@/components/empty-panel";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { DetailSettings, SettingItem } from "@/components/queries/query-ui";
import { QueryGate } from "@/components/query-gate";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useSantaFileAccessEvent } from "@/hooks/use-santa-events";
import { formatDateTime } from "@/lib/utils";

import { fileAccessEventLabel, fileName } from "../events/decisions";
import { FileAccessDecisionBadge, HostLink, Timestamp } from "../events/event-ui";

export function SantaFileAccessEventDetailPage() {
  const { eventId } = useParams({ from: "/_authenticated/santa/events/file-access/$eventId" });
  const id = Number(eventId);
  const query = useSantaFileAccessEvent(Number.isFinite(id) ? id : null);

  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to load file access event"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const event = query.data;
  const processChain = event.process_chain ?? [];

  return (
    <PageShell className="gap-6">
      <PageHeader title={fileAccessEventLabel(event)} description={event.target} />

      <DetailSettings>
        <SettingItem label="Decision">
          <FileAccessDecisionBadge decision={event.decision} />
        </SettingItem>
        <SettingItem label="Host">
          <HostLink host={event.host} />
        </SettingItem>
        <SettingItem label="Rule">{event.rule_name || event.rule_version || "-"}</SettingItem>
        <SettingItem label="Occurred">
          <Timestamp value={event.occurred_at} />
        </SettingItem>
        <SettingItem label="Ingested">{formatDateTime(event.ingested_at)}</SettingItem>
      </DetailSettings>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="process">Process Chain</TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <div className="rounded-md border">
            <Table>
              <TableBody>
                <DetailRow label="Target" value={event.target} />
                <DetailRow label="Rule Name" value={event.rule_name} />
                <DetailRow label="Rule Version" value={event.rule_version} />
                <DetailRow
                  label="Primary Process"
                  value={
                    event.primary_process.file_name || fileName(event.primary_process.file_path)
                  }
                />
              </TableBody>
            </Table>
          </div>
        </TabsContent>

        <TabsContent value="process">
          {processChain.length === 0 ? (
            <EmptyPanel>No process chain</EmptyPanel>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Process</TableHead>
                    <TableHead>PID</TableHead>
                    <TableHead>SHA-256</TableHead>
                    <TableHead>Signing ID</TableHead>
                    <TableHead>Team ID</TableHead>
                    <TableHead>CDHash</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {processChain.map((process) => (
                    <TableRow key={`${process.pid}:${process.file_sha256}:${process.file_path}`}>
                      <TableCell>
                        {process.file_name || fileName(process.file_path) || "-"}
                      </TableCell>
                      <TableCell>{process.pid || "-"}</TableCell>
                      <TableCell className="break-all">{process.file_sha256 || "-"}</TableCell>
                      <TableCell>{process.signing_id || "-"}</TableCell>
                      <TableCell>{process.team_id || "-"}</TableCell>
                      <TableCell className="break-all">{process.cdhash || "-"}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </TabsContent>
      </Tabs>
    </PageShell>
  );
}

function DetailRow({ label, value }: { label: string; value?: string }) {
  return (
    <TableRow>
      <TableCell className="w-48">{label}</TableCell>
      <TableCell>{value ?? "-"}</TableCell>
    </TableRow>
  );
}
