import { Link } from "@tanstack/react-router";

import { EnumStatus } from "@/components/enum-status";
import type {
  SantaExecutionDecision,
  SantaFileAccessDecision,
  SantaHostSummary,
} from "@/hooks/use-santa-events";
import { formatDateTime, formatRelative } from "@/lib/utils";

import { EXECUTION_DECISIONS, FILE_ACCESS_DECISIONS } from "./decisions";

export function ExecutionDecisionBadge({ decision }: { decision: SantaExecutionDecision }) {
  return <EnumStatus value={decision} metadata={EXECUTION_DECISIONS} />;
}

export function FileAccessDecisionBadge({ decision }: { decision: SantaFileAccessDecision }) {
  return <EnumStatus value={decision} metadata={FILE_ACCESS_DECISIONS} />;
}

export function HostLink({ host }: { host: SantaHostSummary }) {
  return (
    <Link
      to="/hosts/$hostId"
      params={{ hostId: String(host.id) }}
      className="font-medium hover:underline"
    >
      {host.display_name}
    </Link>
  );
}

export function Timestamp({ value }: { value: string }) {
  return <span title={formatDateTime(value)}>{formatRelative(value)}</span>;
}
