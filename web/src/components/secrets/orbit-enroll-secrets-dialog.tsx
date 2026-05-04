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
import { useCreateEnrollSecret, useEnrollSecrets } from "@/hooks/use-secrets";

export function OrbitEnrollSecretsDialog({ trigger }: { trigger: React.ReactNode }) {
  const query = useEnrollSecrets();
  const create = useCreateEnrollSecret();
  const data = query.data ?? [];

  return (
    <Dialog>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Manage enroll secrets</DialogTitle>
          <DialogDescription>
            Reusable shared secrets for Orbit/osquery enrollment. Minimum 32 characters. Deleting a secret does not
            invalidate already-enrolled hosts.
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
          emptyTitle="No enroll secrets yet"
          emptyDescription="Create a secret to allow Orbit to enroll Macs against this Woodstar deployment."
        />

        <DialogFooter>
          <Button size="sm" className="gap-2" disabled={create.isPending} onClick={() => create.mutate()}>
            <Plus className="size-4" /> New enroll secret
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
