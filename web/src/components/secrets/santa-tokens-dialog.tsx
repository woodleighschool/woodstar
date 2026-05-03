import { Plus } from "lucide-react";

import { SecretTable } from "@/components/secrets/secret-table";
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
import { useCreateSantaToken, useSantaTokens } from "@/hooks/use-secrets";

export function SantaTokensDialog({
  trigger,
}: {
  trigger: React.ReactNode;
}) {
  const query = useSantaTokens();
  const create = useCreateSantaToken();
  const data = query.data ?? [];

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

        <SecretTable
          data={data.map((row) => ({
            id: row.id,
            value: row.value,
            created_at: row.created_at,
          }))}
          isLoading={query.isLoading}
          error={query.error ?? null}
          onRetry={() => query.refetch()}
          emptyTitle="No Santa sync tokens"
          emptyDescription="Create a token, then deploy it via your Santa configuration profile alongside SyncBaseURL."
        />

        <DialogFooter>
          <Button
            size="sm"
            className="gap-2"
            disabled={create.isPending}
            onClick={() => create.mutate()}
          >
            <Plus className="size-4" /> New sync token
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
