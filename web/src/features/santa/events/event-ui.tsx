import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { TextLink } from "@components/link";
import type { SantaHostSummary } from "@lib/api";
import { formatDateTime, formatRelative } from "@lib/utils";

import {
  EXECUTION_DECISIONS,
  FILE_ACCESS_DECISIONS,
  type SantaExecutionDecision,
  type SantaFileAccessDecision,
} from "./decisions";

export function ExecutionDecisionBadge({ decision }: { decision: SantaExecutionDecision }) {
  return <EnumStatusIndicator value={decision} metadata={EXECUTION_DECISIONS} />;
}

export function FileAccessDecisionBadge({ decision }: { decision: SantaFileAccessDecision }) {
  return <EnumStatusIndicator value={decision} metadata={FILE_ACCESS_DECISIONS} />;
}

export function HostLink({ host }: { host: SantaHostSummary }) {
  return (
    <TextLink to="/hosts/$id" params={{ id: String(host.id) }} className="font-medium">
      {host.display_name}
    </TextLink>
  );
}

export function Timestamp({ value }: { value: string }) {
  return <span title={formatDateTime(value)}>{formatRelative(value)}</span>;
}
