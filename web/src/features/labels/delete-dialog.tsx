import { ConfirmDialog } from "@components/confirm-dialog";
import { toast } from "@components/ui/toast";
import { useDeleteLabel } from "@features/labels/queries";
import type { Label } from "@lib/api";

export function LabelDeleteDialog({
  label,
  open,
  onOpenChange,
  onDeleted,
}: {
  label: Label | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}) {
  const remove = useDeleteLabel();

  async function handleDelete() {
    if (!label) return;
    await remove.mutateAsync(label.id);
    onOpenChange(false);
    toast.add({ title: "Label deleted", type: "success" });
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete label?"
      description={
        label
          ? `${label.name} will be removed from hosts and filters.`
          : "This label will be removed."
      }
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleDelete()}
    />
  );
}
