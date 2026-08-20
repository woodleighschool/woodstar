import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { QueryError } from "@components/query-error";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@components/ui/dialog";
import { Skeleton } from "@components/ui/skeleton";

import { usePolicyRemediationRun } from "./queries";
import { PolicyRemediationStatus } from "./query-results";

export function PolicyRemediationDialog({
  policyID,
  hostID,
  hostName,
  open,
  onOpenChange,
}: {
  policyID: number;
  hostID: number | null;
  hostName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const query = usePolicyRemediationRun(open ? policyID : null, open ? hostID : null);
  const run = query.data;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100vh-2rem)] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Remediation</DialogTitle>
          <DialogDescription>{hostName}</DialogDescription>
        </DialogHeader>

        {query.error ? (
          <QueryError
            title="Failed to Load Remediation"
            error={query.error}
            onRetry={() => void query.refetch()}
          />
        ) : !run ? (
          <div className="flex flex-col gap-3">
            <Skeleton className="h-8 w-32" />
            <Skeleton className="h-40 w-full" />
          </div>
        ) : (
          <div className="flex min-w-0 flex-col gap-5">
            <KeyValueSection title="Execution">
              <KeyValueRow label="Status" value={<PolicyRemediationStatus status={run.status} />} />
              <KeyValueRow label="Started By" value={run.automatic ? "Automatic" : "Manual"} />
              <KeyValueRow
                label="Runtime"
                value={run.runtime_seconds === undefined ? "-" : `${run.runtime_seconds}s`}
              />
              <KeyValueRow
                label="Exit Code"
                value={run.exit_code === undefined ? "-" : String(run.exit_code)}
              />
            </KeyValueSection>

            <section className="flex min-w-0 flex-col gap-3">
              <h3 className="text-sm font-medium text-foreground">Output</h3>
              <pre className="max-h-80 overflow-auto rounded-lg bg-muted p-3 font-mono text-xs whitespace-pre-wrap">
                {run.output || "No Output"}
              </pre>
            </section>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
