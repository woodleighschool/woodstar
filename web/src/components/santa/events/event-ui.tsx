import { Link } from "@tanstack/react-router";

import { EnumBadge } from "@/components/enum-badge";
import type { SantaExecutionDecision, SantaFileAccessDecision, SantaHostSummary } from "@/hooks/use-santa";
import { EXECUTION_DECISIONS, FILE_ACCESS_DECISIONS } from "@/lib/santa-events";
import { formatRelative } from "@/lib/utils";

export function ExecutionDecisionBadge({ decision }: { decision: SantaExecutionDecision }) {
  return <EnumBadge value={decision} metadata={EXECUTION_DECISIONS} />;
}

export function FileAccessDecisionBadge({ decision }: { decision: SantaFileAccessDecision }) {
  return <EnumBadge value={decision} metadata={FILE_ACCESS_DECISIONS} />;
}

export function HostLink({ host }: { host: SantaHostSummary }) {
  return (
    <Link to="/hosts/$hostId" params={{ hostId: String(host.id) }} className="font-medium hover:underline">
      {host.display_name}
    </Link>
  );
}

export function Timestamp({ value }: { value: string }) {
  return <span title={new Date(value).toLocaleString()}>{formatRelative(value)}</span>;
}
