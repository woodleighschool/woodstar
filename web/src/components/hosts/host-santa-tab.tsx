import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Activity, FolderLock } from "lucide-react";
import { useMemo } from "react";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { EmptyPanel } from "@/components/empty-panel";
import { KeyValueGrid, KeyValueItem } from "@/components/key-value";
import { QueryError } from "@/components/query-error";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useHostSantaRules } from "@/hooks/use-hosts";
import type { SantaHostState, SantaRuleStatus } from "@/lib/api";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { clientModeLabel } from "@/lib/santa-configurations";
import { policyLabel, ruleTypeLabel } from "@/lib/santa-rules";
import { formatRelative } from "@/lib/utils";
export function HostSantaTab({ hostId, santa }: { hostId: number; santa: SantaHostState }) {
  const rules = useHostSantaRules(hostId, { per_page: MAX_PAGE_SIZE });
  const items = rules.data?.items ?? [];
  const totalCount = rules.data?.count ?? 0;
  const matchedLabelName = santa?.configuration?.matched_via_label?.name;
  const configurationValue = santa?.configuration?.name ? (
    matchedLabelName ? (
      <Tooltip>
        <TooltipTrigger render={<span />}>{santa.configuration.name}</TooltipTrigger>
        <TooltipContent>
          <div>{`Matched via label: ${matchedLabelName}`}</div>
        </TooltipContent>
      </Tooltip>
    ) : (
      santa.configuration.name
    )
  ) : undefined;
  const columns = useMemo<ColumnDef<SantaRuleStatus>[]>(
    () => [
      {
        accessorKey: "name",
        header: () => "Name",
        cell: ({ row }) => row.original.name,
      },
      {
        accessorKey: "identifier",
        header: () => "Identifier",
        cell: ({ row }) => row.original.identifier,
      },
      {
        accessorKey: "rule_type",
        header: () => "Type",
        cell: ({ row }) => ruleTypeLabel(row.original.rule_type),
      },
      {
        accessorKey: "policy",
        header: () => "Policy",
        cell: ({ row }) => policyLabel(row.original.policy),
      },
      {
        accessorKey: "applied",
        header: () => "Status",
        cell: ({ row }) => (
          <Badge variant={!row.original.applied ? "secondary" : "outline"} className="gap-1.5">
            <span
              className={
                !row.original.applied
                  ? "size-1.5 rounded-full bg-warning"
                  : "size-1.5 rounded-full bg-status-online"
              }
            />
            {row.original.applied ? "Applied" : "Pending"}
          </Badge>
        ),
      },
    ],
    [],
  );
  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent>
          <div className="mb-5 flex flex-wrap gap-2">
            <Button
              variant="outline"
              size="sm"
              render={<Link to="/santa/events" search={{ host_id: hostId }} />}
              nativeButton={false}
            >
              <Activity data-icon="inline-start" />
              View Execution Events
            </Button>
            <Button
              variant="outline"
              size="sm"
              render={<Link to="/santa/events/file-access" search={{ host_id: hostId }} />}
              nativeButton={false}
            >
              <FolderLock data-icon="inline-start" />
              View File Access Events
            </Button>
          </div>
          <KeyValueGrid>
            <KeyValueItem label="Version" value={santa.version} />
            <KeyValueItem label="Client Mode" value={clientModeLabel(santa.client_mode_reported)} />
            <KeyValueItem label="Configuration" value={configurationValue} />
            <KeyValueItem label="Last Sync" value={formatRelative(santa.last_seen_at)} />
            <KeyValueItem
              label="Rule Sync"
              value={`${santa.rule_sync.applied_count} applied / ${santa.rule_sync.desired_count} desired`}
            />
            <KeyValueItem label="Pending Rules" value={santa.rule_sync.pending_count} />
          </KeyValueGrid>
        </CardContent>
      </Card>

      <Card className="gap-4 py-4">
        <CardHeader className="flex flex-row items-center justify-between gap-3">
          <CardTitle>Rules</CardTitle>
        </CardHeader>
        <CardContent>
          {rules.error ? (
            <QueryError
              title="Failed to load rules"
              error={rules.error}
              onRetry={() => void rules.refetch()}
            />
          ) : rules.isLoading ? null : totalCount === 0 ? (
            <EmptyPanel>No matching rules</EmptyPanel>
          ) : (
            <DataTableStatic columns={columns} data={items} />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
