import { Link } from "@tanstack/react-router";

import { Badge } from "@/components/ui/badge";
import type { SantaHostSummary } from "@/hooks/use-santa";
import { formatRelative } from "@/lib/utils";

export function DecisionBadge({ decision }: { decision: string }) {
  const blocked = decision.startsWith("block_");
  const allowed = decision.startsWith("allow_") || decision === "audit_only";
  return (
    <Badge variant={blocked ? "destructive" : allowed ? "outline" : "secondary"} className="gap-1.5">
      {allowed ? <span className="bg-status-online size-1.5 rounded-full" /> : null}
      {decision.replaceAll("_", " ")}
    </Badge>
  );
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
