import { Play } from "lucide-react";
import { useMemo, useState } from "react";

import { ConfirmDialog } from "@components/confirm-dialog";
import type { DataTableInstance } from "@components/data-table/types";
import {
  ActionBar,
  ActionBarGroup,
  ActionBarItem,
  ActionBarSelection,
  ActionBarSeparator,
} from "@components/ui/action-bar";

import { useRunPolicyRemediations } from "./queries";
import type { PolicyResultRow } from "./query-results";

export function PolicyResultsActionBar({
  table,
  policyID,
}: {
  table: DataTableInstance<PolicyResultRow>;
  policyID: number;
}) {
  const rows = table.getFilteredSelectedRowModel().rows;
  const hostIDs = useMemo(() => rows.map((row) => row.original.host_id), [rows]);
  const [open, setOpen] = useState(false);
  const runRemediations = useRunPolicyRemediations();

  return (
    <>
      <ActionBar
        open={hostIDs.length > 0}
        onOpenChange={(next) => {
          if (!next) table.toggleAllRowsSelected(false);
        }}
      >
        <ActionBarSelection>{hostIDs.length} Selected</ActionBarSelection>
        <ActionBarSeparator />
        <ActionBarGroup>
          <ActionBarItem
            onSelect={(event) => {
              event.preventDefault();
              setOpen(true);
            }}
            disabled={runRemediations.isPending}
          >
            <Play data-icon="inline-start" />
            Run Remediation
          </ActionBarItem>
        </ActionBarGroup>
      </ActionBar>
      <ConfirmDialog
        open={open}
        onOpenChange={(next) => {
          if (!next && !runRemediations.isPending) {
            runRemediations.reset();
            setOpen(false);
          }
        }}
        title={`Run Remediation for ${hostIDs.length} Selected ${hostIDs.length === 1 ? "Host" : "Hosts"}?`}
        description="Each eligible host receives the policy's current script. Hosts with remediation already queued or in progress are skipped."
        confirmLabel="Run Remediation"
        pending={runRemediations.isPending}
        onConfirm={() => {
          runRemediations.mutate(
            { policyID, hostIDs },
            {
              onSuccess: () => {
                table.toggleAllRowsSelected(false);
                setOpen(false);
              },
            },
          );
        }}
      />
    </>
  );
}
