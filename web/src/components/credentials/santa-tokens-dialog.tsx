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
import type { SantaToken } from "@/lib/types";

export function SantaTokensDialog({
  trigger,
}: {
  trigger: React.ReactNode;
}) {
  const { data, query, isPending } = useResourceList<SantaToken>(
    endpoints.santaTokens,
    queryKeys.santaTokens,
  );

  return (
    <Dialog>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Manage Santa sync tokens</DialogTitle>
          <DialogDescription>
            Bearer tokens delivered via Santa configuration profile. Tokens gate
            sync access; host identity still comes from the Santa payload.
          </DialogDescription>
        </DialogHeader>

        <CredentialTable
          endpoint={endpoints.santaTokens}
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
          emptyTitle="No Santa sync tokens"
          emptyDescription="Create a token, then deploy it via your Santa configuration profile alongside SyncBaseURL."
        />

        <DialogFooter>
          <Button size="sm" disabled={isPending} className="gap-2">
            <Plus className="size-4" /> New sync token
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
