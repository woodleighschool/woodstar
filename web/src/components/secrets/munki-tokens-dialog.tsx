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
import { useCreateMunkiToken, useMunkiTokens } from "@/hooks/use-secrets";

export function MunkiTokensDialog({
  trigger,
}: {
  trigger: React.ReactNode;
}) {
  const query = useMunkiTokens();
  const create = useCreateMunkiToken();
  const data = query.data ?? [];

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

        <SecretTable
          data={data.map((row) => ({
            id: row.id,
            value: row.value,
            created_at: row.created_at,
          }))}
          isLoading={query.isLoading}
          error={query.error ?? null}
          onRetry={() => query.refetch()}
          emptyTitle="No Munki repo tokens"
          emptyDescription="Create a token, then ship it inside ManagedInstalls AdditionalHttpHeaders alongside the serial header."
        />

        <DialogFooter>
          <Button
            size="sm"
            className="gap-2"
            disabled={create.isPending}
            onClick={() => create.mutate()}
          >
            <Plus className="size-4" /> New repo token
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
