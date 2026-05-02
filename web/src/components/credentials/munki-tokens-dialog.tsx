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
import type { MunkiToken } from "@/lib/types";

export function MunkiTokensDialog({
  trigger,
}: {
  trigger: React.ReactNode;
}) {
  const { data, query, isPending } = useResourceList<MunkiToken>(
    endpoints.munkiTokens,
    queryKeys.munkiTokens,
  );

  return (
    <Dialog>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Manage Munki repo tokens</DialogTitle>
          <DialogDescription>
            Bearer tokens delivered via Munki AdditionalHttpHeaders. Munki
            requests must also include an MDM-expanded serial header to bind to
            an existing Woodstar host.
          </DialogDescription>
        </DialogHeader>

        <CredentialTable
          endpoint={endpoints.munkiTokens}
          data={data.map((row) => ({
            id: row.id,
            label: row.label,
            preview: row.preview,
            created_at: row.created_at,
            last_used_at: row.last_used_at,
          }))}
          isPending={isPending}
          isLoading={query.isLoading}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyTitle="No Munki repo tokens"
          emptyDescription="Create a token, then ship it inside ManagedInstalls AdditionalHttpHeaders alongside the serial header."
        />

        <DialogFooter>
          <Button size="sm" disabled={isPending} className="gap-2">
            <Plus className="size-4" /> New repo token
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
