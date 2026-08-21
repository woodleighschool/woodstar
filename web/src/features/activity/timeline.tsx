import { CircleDot } from "lucide-react";
import type { ReactNode } from "react";

import { Link } from "@components/link";
import { RelativeTime } from "@components/relative-time";
import { Badge } from "@components/ui/badge";
import type { ActivityEvent } from "@lib/api";
import { assertNever, nonEmpty } from "@lib/utils";

export function ActivityTimeline({
  events,
  showArea = false,
}: {
  events: ActivityEvent[];
  showArea?: boolean;
}) {
  return (
    <ol className="relative ml-2 border-l border-border">
      {events.map((event) => (
        <li key={event.id} className="relative pb-5 pl-6 last:pb-0">
          <span className="absolute top-0.5 -left-2 flex size-4 items-center justify-center rounded-full bg-background text-muted-foreground">
            <CircleDot className="size-3.5" />
          </span>
          <div className="flex min-w-0 flex-col gap-1">
            <p className="min-w-0 text-sm/relaxed">
              <span className="font-medium text-foreground">{actorLabel(event)}</span>{" "}
              {actionLabel(event.action)} {subjectLabel(event)}
            </p>
            <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <RelativeTime value={event.occurred_at} />
              {showArea ? <Badge variant="secondary">{areaLabel(event.area)}</Badge> : null}
            </div>
          </div>
        </li>
      ))}
    </ol>
  );
}

function actorLabel(event: ActivityEvent): string {
  if (event.actor.kind === "system") return "System";
  return nonEmpty(event.actor.name) ?? nonEmpty(event.actor.email) ?? "Administrator";
}

function actionLabel(action: ActivityEvent["action"]): string {
  switch (action) {
    case "orbit_host_enrolled":
      return "enrolled through Orbit";
    case "osquery_host_enrolled":
      return "enrolled through osquery";
    case "host_deleted":
    case "hosts_deleted":
    case "policy_deleted":
    case "policies_deleted":
    case "report_deleted":
    case "reports_deleted":
      return "deleted";
    case "host_inventory_requested":
      return "requested fresh inventory from";
    case "host_primary_user_set":
      return "set the primary user for";
    case "host_primary_user_cleared":
      return "cleared the primary user from";
    case "policy_created":
    case "report_created":
      return "created";
    case "policy_updated":
    case "report_updated":
      return "updated";
    case "policy_remediation_requested":
      return "requested remediation for";
    case "live_query_started":
      return "started a live query for";
    case "live_query_stopped":
      return "stopped";
    default:
      return assertNever(action);
  }
}

function subjectLabel(event: ActivityEvent): ReactNode {
  const label = nonEmpty(event.subject.name) ?? subjectTypeLabel(event.subject.type);
  const id = event.subject.id;
  if (id === undefined || event.action.includes("deleted")) return label;

  switch (event.subject.type) {
    case "host":
      return (
        <Link to="/hosts/$id" params={{ id: String(id) }} className="font-medium hover:underline">
          {label}
        </Link>
      );
    case "policy":
      return (
        <Link
          to="/osquery/policies/$id"
          params={{ id: String(id) }}
          className="font-medium hover:underline"
        >
          {label}
        </Link>
      );
    case "report":
      return (
        <Link
          to="/osquery/reports/$id"
          params={{ id: String(id) }}
          className="font-medium hover:underline"
        >
          {label}
        </Link>
      );
    default:
      return label;
  }
}

function subjectTypeLabel(type: string): string {
  return type.replaceAll("_", " ");
}

function areaLabel(area: ActivityEvent["area"]): string {
  return area === "osquery" ? "osquery" : "Hosts";
}
