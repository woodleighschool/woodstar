import { useParams, useSearch } from "@tanstack/react-router";

import { TableSurface } from "@components/data-table/table-surface";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@components/layout/scrollable-tabs";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
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
import { formatDateTime } from "@lib/utils";

import { FileAccessDecisionBadge, HostLink, Timestamp } from "./event-ui";
import { useSantaFileAccessEvent } from "./queries";

export function SantaFileAccessEventDetailPage() {
  const { id: eventId } = useParams({
    from: "/_authenticated/santa/events/file-access/$id",
  });
  const search = useSearch({
    from: "/_authenticated/santa/events/file-access/$id",
  });
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
  const activeTab = search.tab === "process-chain" ? "process-chain" : "overview";

  return (
    <PageShell className="gap-6">
      <PageHeader title="File Access" description={event.target} />

      <ScrollableTabs value={activeTab}>
        <ScrollableTabsList>
          <TabsTrigger
            value="overview"
            render={
              <Link
                to="/santa/events/file-access/$id"
                params={{ id: eventId }}
                search={{ ...search, tab: undefined }}
              />
            }
            nativeButton={false}
          >
            Overview
          </TabsTrigger>
          <TabsTrigger
            value="process-chain"
            render={
              <Link
                to="/santa/events/file-access/$id"
                params={{ id: eventId }}
                search={{ ...search, tab: "process-chain" }}
              />
            }
            nativeButton={false}
          >
            Process Chain
          </TabsTrigger>
        </ScrollableTabsList>

        <TabsContent value="overview" className="flex flex-col gap-5">
          <KeyValueSection title="Overview">
            <KeyValueRow
              label="Decision"
              value={<FileAccessDecisionBadge decision={event.decision} />}
            />
            <KeyValueRow label="Host" value={<HostLink host={event.host} />} />
            <KeyValueRow label="Rule Name" value={event.rule_name} />
            <KeyValueRow label="Rule Version" value={event.rule_version} />
            <KeyValueRow label="Occurred" value={<Timestamp value={event.occurred_at} />} />
            <KeyValueRow label="Ingested" value={formatDateTime(event.ingested_at)} />
          </KeyValueSection>

          <KeyValueSection title="Process">
            <KeyValueRow label="Target" value={event.target} />
            <KeyValueRow label="Primary Process" value={event.primary_process.file_name} />
          </KeyValueSection>
        </TabsContent>

        <TabsContent value="process-chain">
          {processChain.length === 0 ? (
            <PanelEmptyState>No Process Chain</PanelEmptyState>
          ) : (
            <TableSurface>
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
                      <TableCell>{process.file_name || "-"}</TableCell>
                      <TableCell>{process.pid}</TableCell>
                      <TableCell className="break-all">{process.file_sha256 || "-"}</TableCell>
                      <TableCell>{process.signing_id || "-"}</TableCell>
                      <TableCell>{process.team_id || "-"}</TableCell>
                      <TableCell className="break-all">{process.cdhash || "-"}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableSurface>
          )}
        </TabsContent>
      </ScrollableTabs>
    </PageShell>
  );
}
