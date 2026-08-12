import { Activity } from "lucide-react";

import { BooleanIndicator } from "@components/boolean-indicator";
import { DataTableStatic } from "@components/data-table/data-table-static";
import type { DataTableColumnDef } from "@components/data-table/types";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { Link, TextLink } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import { Separator } from "@components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@components/ui/tooltip";
import { useHostSantaRules } from "@features/hosts/queries";
import { CLIENT_MODES } from "@features/santa/configurations/metadata";
import { POLICIES, ruleTypeLabel } from "@features/santa/rules/metadata";
import type { ApiError, SantaHostState, SantaRuleStatus } from "@lib/api";
import { MAX_PAGE_SIZE } from "@lib/pagination";

const santaRuleColumns: DataTableColumnDef<SantaRuleStatus>[] = [
  {
    accessorKey: "name",
    header: () => "Name",
    cell: ({ row }) => (
      <TextLink
        to="/santa/rules/$id"
        params={{ id: String(row.original.rule_id) }}
        className="font-medium"
      >
        {row.original.name}
      </TextLink>
    ),
    size: 240,
    minSize: 160,
  },
  {
    accessorKey: "rule_type",
    header: () => "Rule Type",
    cell: ({ row }) => ruleTypeLabel(row.original.rule_type),
    size: 112,
    minSize: 112,
    maxSize: 112,
    enableResizing: false,
  },
  {
    accessorKey: "identifier",
    header: () => "Identifier",
    cell: ({ row }) => row.original.identifier || "-",
    size: 320,
    minSize: 200,
  },
  {
    accessorKey: "policy",
    header: () => "Policy",
    cell: ({ row }) => POLICIES[row.original.policy].name,
    size: 104,
    minSize: 104,
    maxSize: 104,
    enableResizing: false,
  },
  {
    accessorKey: "applied",
    header: () => "Applied",
    cell: ({ row }) => (
      <BooleanIndicator
        value={row.original.applied}
        trueLabel="Applied"
        falseLabel="Pending"
        tone="positive"
      />
    ),
    size: 80,
    minSize: 80,
    maxSize: 80,
    enableResizing: false,
  },
];

interface HostSantaTabProps {
  hostId: number;
  santa: SantaHostState | null | undefined;
  stateError: ApiError | null;
  onStateRetry: () => void;
}

export function HostSantaTab({ hostId, santa, stateError, onStateRetry }: HostSantaTabProps) {
  const rules = useHostSantaRules(hostId, { per_page: MAX_PAGE_SIZE });
  const items = rules.data?.items ?? [];
  const configuration = santa?.configuration;
  return (
    <div className="flex flex-col gap-4">
      {stateError ? (
        <QueryError title="Failed to load Santa state" error={stateError} onRetry={onStateRetry} />
      ) : santa === null ? (
        <PanelEmptyState>No Santa sync reported</PanelEmptyState>
      ) : santa ? (
        <KeyValueSection
          title="Overview"
          actions={
            <Button
              variant="outline"
              size="sm"
              render={<Link to="/santa/events" search={{ host_id: hostId }} />}
              nativeButton={false}
            >
              <Activity data-icon="inline-start" />
              See Events
            </Button>
          }
        >
          <KeyValueRow label="Version" value={santa.version} />
          <KeyValueRow label="Client Mode" value={CLIENT_MODES[santa.client_mode_reported].name} />
          <KeyValueRow
            label="Configuration"
            value={configuration ? <SantaConfigurationLink configuration={configuration} /> : null}
          />
          <KeyValueRow
            label="Rule Sync"
            value={`${santa.rule_sync.applied_count} applied / ${santa.rule_sync.desired_count} desired`}
          />
        </KeyValueSection>
      ) : null}

      {rules.error ? (
        <section className="flex min-w-0 flex-col gap-3">
          <h2 className="text-base/snug font-medium text-foreground">Rules</h2>
          <Separator />
          <div className="px-3">
            <QueryError
              title="Failed to load rules"
              error={rules.error}
              onRetry={() => void rules.refetch()}
            />
          </div>
        </section>
      ) : rules.isLoading ? null : (
        <DataTableStatic
          heading="Rules"
          columns={santaRuleColumns}
          data={items}
          empty={<PanelEmptyState>No matching rules</PanelEmptyState>}
        />
      )}
    </div>
  );
}

function SantaConfigurationLink({
  configuration,
}: {
  configuration: NonNullable<SantaHostState["configuration"]>;
}) {
  const link = (
    <TextLink
      to="/santa/configurations/$id"
      params={{ id: String(configuration.id) }}
      className="font-medium"
    >
      {configuration.name}
    </TextLink>
  );

  return configuration.matched_via_label ? (
    <Tooltip>
      <TooltipTrigger render={link} />
      <TooltipContent>
        <div>{`Matched via label: ${configuration.matched_via_label.name}`}</div>
      </TooltipContent>
    </Tooltip>
  ) : (
    link
  );
}
