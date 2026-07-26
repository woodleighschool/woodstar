import { toast } from "sonner";

import { ConfirmDialog } from "@components/confirm-dialog";
import type { SantaRule } from "@lib/api";

import { useDeleteSantaRule } from "./queries";

export function RuleDeleteDialog({
  rule,
  open,
  onOpenChange,
  onDeleted,
}: {
  rule: SantaRule | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}) {
  const remove = useDeleteSantaRule();

  async function handleDelete() {
    if (!rule) return;
    await remove.mutateAsync(rule.id);
    onOpenChange(false);
    toast.success("Rule deleted");
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete rule?"
      description={
        rule
          ? `${rule.name} will stop syncing to targeted Santa clients.`
          : "This rule will stop syncing."
      }
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleDelete()}
    />
  );
}
