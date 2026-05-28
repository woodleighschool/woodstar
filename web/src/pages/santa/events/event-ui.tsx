import { Link } from "@tanstack/react-router";

import { Badge } from "@/components/ui/badge";
import type { SantaHostSummary } from "@/hooks/use-santa";
import { formatRelative } from "@/lib/utils";

const DECISION_WORDS: Record<string, string> = {
  cdhash: "CDHash",
  id: "ID",
  teamid: "Team ID",
};

export function DecisionBadge({ decision }: { decision: string }) {
  const blocked = decision.startsWith("block_");
  const allowed = decision.startsWith("allow_") || decision === "audit_only";
  return (
    <Badge variant={blocked ? "destructive" : allowed ? "outline" : "secondary"} className="gap-1.5">
      {allowed ? <span className="bg-status-online size-1.5 rounded-full" /> : null}
      {decisionLabel(decision)}
    </Badge>
  );
}

function decisionLabel(decision: string) {
  return decision
    .split("_")
    .map((word) => DECISION_WORDS[word] ?? word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

export function HostLink({ host }: { host: SantaHostSummary }) {
  return (
    <Link to="/hosts/$hostId" params={{ hostId: String(host.id) }} className="font-medium hover:underline">
      {host.display_name || host.hostname || host.computer_name || host.hardware_serial || host.id}
    </Link>
  );
}

export function Timestamp({ value }: { value: string }) {
  return <span title={new Date(value).toLocaleString()}>{formatRelative(value)}</span>;
}
