import { Eye, MoreHorizontal, Play } from "lucide-react";

import type { DataTableColumnDef } from "@components/data-table/types";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { TextLink } from "@components/link";
import { Button } from "@components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@components/ui/dropdown-menu";
import type { OsqueryPolicyHostStatus, OsqueryPolicyRemediationRunSummary } from "@lib/api";
import { formatRelative } from "@lib/utils";

import {
  POLICY_RESULT_STATUSES,
  REMEDIATION_RUN_STATUSES,
  type PolicyResultDisplayStatus,
  type RemediationRunStatus,
} from "./model";

export type PolicyResultRow = {
  host_id: number;
  host_name: string;
  status: PolicyResultDisplayStatus;
  updated_at?: string;
  error?: string;
  remediation?: OsqueryPolicyRemediationRunSummary;
  onRunRemediation?: () => void;
  onViewRemediation?: () => void;
};

export function policyResultFromStatus(
  result: OsqueryPolicyHostStatus,
  actions: Pick<PolicyResultRow, "onRunRemediation" | "onViewRemediation"> = {},
): PolicyResultRow {
  return {
    host_id: result.host_id,
    host_name: result.host_name,
    status: result.status,
    updated_at: result.updated_at,
    error: result.error,
    remediation: result.remediation,
    ...actions,
  };
}

export function createPolicyResultColumns({
  timestampHeader,
  includeError = false,
  includeRemediation = false,
  includeActions = false,
}: {
  timestampHeader: "Last Evaluated";
  includeError?: boolean;
  includeRemediation?: boolean;
  includeActions?: boolean;
}): DataTableColumnDef<PolicyResultRow>[] {
  const columns: DataTableColumnDef<PolicyResultRow>[] = [
    {
      accessorKey: "host_name",
      header: () => "Host",
      cell: ({ row }) => (
        <TextLink
          to="/hosts/$id"
          params={{ id: String(row.original.host_id) }}
          className="font-medium"
        >
          {row.original.host_name}
        </TextLink>
      ),
    },
    {
      accessorKey: "status",
      header: () => "Status",
      enableColumnFilter: true,
      cell: ({ row }) => <PolicyResultStatus status={row.original.status} />,
    },
    {
      accessorKey: "updated_at",
      header: () => timestampHeader,
      cell: ({ row }) => (row.original.updated_at ? formatRelative(row.original.updated_at) : "-"),
    },
  ];
  if (includeError) {
    columns.push({
      accessorKey: "error",
      header: () => "Error",
      cell: ({ row }) => row.original.error || "-",
    });
  }
  if (includeRemediation) {
    columns.push({
      id: "remediation",
      header: () => "Remediation",
      cell: ({ row }) =>
        row.original.remediation ? (
          <PolicyRemediationStatus status={row.original.remediation.status} />
        ) : (
          "-"
        ),
    });
  }
  if (includeActions) {
    columns.push({
      id: "actions",
      header: () => null,
      enableSorting: false,
      enableHiding: false,
      size: 44,
      minSize: 44,
      maxSize: 44,
      enableResizing: false,
      cell: PolicyResultActionsCell,
    });
  }
  return columns;
}

function PolicyResultActionsCell({ row }: { row: { original: PolicyResultRow } }) {
  const result = row.original;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button type="button" size="icon" variant="ghost" />}>
        <MoreHorizontal />
        <span className="sr-only">Open policy result actions</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-48">
        <DropdownMenuGroup>
          <DropdownMenuItem
            disabled={!result.remediation || !result.onViewRemediation}
            onClick={result.onViewRemediation}
          >
            <Eye />
            View Remediation
          </DropdownMenuItem>
          {result.onRunRemediation ? (
            <DropdownMenuItem disabled={result.status !== "fail"} onClick={result.onRunRemediation}>
              <Play />
              Run Remediation
            </DropdownMenuItem>
          ) : null}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function policyResultStatusLabel(status: PolicyResultDisplayStatus) {
  return POLICY_RESULT_STATUSES[status].name;
}

export function PolicyResultStatus({ status }: { status: PolicyResultDisplayStatus }) {
  return <EnumStatusIndicator value={status} metadata={POLICY_RESULT_STATUSES} />;
}

export function PolicyRemediationStatus({ status }: { status: RemediationRunStatus }) {
  return <EnumStatusIndicator value={status} metadata={REMEDIATION_RUN_STATUSES} />;
}
