import { Plus } from "lucide-react";

import { CredentialTable } from "@/components/credentials/credential-table";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { useResourceList } from "@/hooks/use-pending-list";
import { endpoints } from "@/lib/endpoints";
import { queryKeys } from "@/lib/query-keys";
import type { EnrollSecret } from "@/lib/types";

export function OrbitEnrollSecretsDialog({
  trigger,
}: {
  trigger: React.ReactNode;
}) {
  const { data, query, isPending } = useResourceList<EnrollSecret>(
    endpoints.enrollSecrets,
    queryKeys.enrollSecrets,
  );

  return (
    <Dialog>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Manage enroll secrets</DialogTitle>
          <DialogDescription>
            Reusable shared secrets for Orbit/osquery enrollment. Minimum 32
            characters. Deleting a secret does not invalidate already-enrolled
            hosts.
          </DialogDescription>
        </DialogHeader>

        <CredentialTable
          endpoint={endpoints.enrollSecrets}
          data={data.map((row) => ({
            id: row.id,
            preview: row.secret_preview,
            created_at: row.created_at,
            last_used_at: row.rotated_at,
            trailing: (
              <span className="text-xs text-muted-foreground">
                {row.host_count} hosts
              </span>
            ),
          }))}
          isPending={isPending}
          isLoading={query.isLoading}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyTitle="No enroll secrets yet"
          emptyDescription="Create a secret to allow Orbit to enroll Macs against this Woodstar deployment."
          trailingHeader="Hosts"
        />

        <DialogFooter>
          <Button size="sm" disabled={isPending} className="gap-2">
            <Plus className="size-4" /> New enroll secret
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
