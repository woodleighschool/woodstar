import { toast } from "sonner";

import { ConfirmDialog } from "@components/confirm-dialog";
import type { OsqueryCheck } from "@lib/api";

import { useDeleteCheck } from "./queries";

export interface CheckDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  check: OsqueryCheck | null;
  onDeleted?: () => void;
}

export function CheckDeleteDialog({
  open,
  onOpenChange,
  check,
  onDeleted,
}: CheckDeleteDialogProps) {
  const remove = useDeleteCheck();

  async function handleConfirm() {
    if (!check) return;
    await remove.mutateAsync(check.id);
    onOpenChange(false);
    toast.success("Check deleted");
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) remove.reset();
        onOpenChange(next);
      }}
      title="Delete Check"
      description={
        check
          ? `${check.name} will be permanently deleted.`
          : "This check will be permanently deleted."
      }
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleConfirm()}
    />
  );
}
