import { ConfirmDialog } from "@components/confirm-dialog";
import { toast } from "@components/ui/toast";
import type { OsqueryPolicy } from "@lib/api";

import { useDeletePolicy } from "./queries";

export interface PolicyDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  policy: OsqueryPolicy | null;
  onDeleted?: () => void;
}

export function PolicyDeleteDialog({
  open,
  onOpenChange,
  policy,
  onDeleted,
}: PolicyDeleteDialogProps) {
  const remove = useDeletePolicy();

  async function handleConfirm() {
    if (!policy) return;
    await remove.mutateAsync(policy.id);
    onOpenChange(false);
    toast.add({ title: "Policy deleted", type: "success" });
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) remove.reset();
        onOpenChange(next);
      }}
      title="Delete Policy"
      description={
        policy
          ? `${policy.name} will be permanently deleted.`
          : "This policy will be permanently deleted."
      }
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleConfirm()}
    />
  );
}
