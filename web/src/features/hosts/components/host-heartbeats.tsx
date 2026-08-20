import { DataTableStatic } from "@components/data-table/data-table-static";
import type { DataTableColumnDef } from "@components/data-table/types";
import { PanelEmptyState } from "@components/panel-empty-state";
import { RelativeTime } from "@components/relative-time";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@components/ui/hover-card";
import type { Heartbeat, Host } from "@lib/api";

const heartbeatSourceLabels: Record<string, string> = {
  orbit: "Orbit",
  osquery: "osquery",
  munki: "Munki",
  santa: "Santa",
};

const heartbeatSourceRanks: Record<string, number> = {
  orbit: 0,
  osquery: 1,
  munki: 2,
  santa: 3,
};

const heartbeatColumns: DataTableColumnDef<Heartbeat>[] = [
  {
    accessorKey: "source",
    header: () => "Agent",
    cell: ({ row }) => heartbeatSourceLabel(row.original.source),
  },
  {
    accessorKey: "last_seen_at",
    header: () => "Last Seen",
    cell: ({ row }) => <HeartbeatTime value={row.original.last_seen_at} />,
  },
  {
    accessorKey: "remote_ip",
    header: () => "Remote IP",
    cell: ({ row }) => row.original.remote_ip ?? "-",
  },
  {
    accessorKey: "user_agent",
    header: () => "User Agent",
    cell: ({ row }) => (
      <span className="block max-w-64 truncate" title={row.original.user_agent}>
        {row.original.user_agent || "-"}
      </span>
    ),
  },
];

export function HostLastContact({ host }: { host: Host }) {
  const heartbeats = orderedHeartbeats(host.heartbeats);

  if (!host.last_contact) return "Never";

  if (heartbeats.length === 0) return <RelativeTime value={host.last_contact} />;

  return (
    <HoverCard>
      <HoverCardTrigger
        render={
          <button
            type="button"
            className="cursor-default underline decoration-dotted underline-offset-4"
          >
            <RelativeTime value={host.last_contact} />
          </button>
        }
      />
      <HoverCardContent align="start" className="w-72">
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
          {heartbeats.map((heartbeat) => (
            <div key={heartbeat.source} className="contents">
              <dt className="text-muted-foreground">{heartbeatSourceLabel(heartbeat.source)}</dt>
              <dd className="text-right">
                <HeartbeatTime value={heartbeat.last_seen_at} />
              </dd>
            </div>
          ))}
        </dl>
      </HoverCardContent>
    </HoverCard>
  );
}

export function HostHeartbeatTable({ heartbeats }: { heartbeats: Heartbeat[] }) {
  return (
    <DataTableStatic
      heading="Heartbeats"
      columns={heartbeatColumns}
      data={orderedHeartbeats(heartbeats)}
      empty={<PanelEmptyState>No Agent Contact Reported</PanelEmptyState>}
    />
  );
}

function HeartbeatTime({ value }: { value: string }) {
  return <RelativeTime value={value} />;
}

function heartbeatSourceLabel(source: string) {
  return heartbeatSourceLabels[source] ?? source;
}

function orderedHeartbeats(heartbeats: Heartbeat[]) {
  return heartbeats.toSorted((left, right) => sourceRank(left.source) - sourceRank(right.source));
}

function sourceRank(source: string) {
  return heartbeatSourceRanks[source] ?? Number.MAX_SAFE_INTEGER;
}
