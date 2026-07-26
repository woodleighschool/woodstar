import { toast } from "sonner";

import { ConfirmDialog } from "@components/confirm-dialog";
import type { MunkiDistributionPoint } from "@lib/api";

import { useDeleteMunkiDistributionPoint } from "./queries";

export function DistributionPointDeleteDialog({
  point,
  open,
  onOpenChange,
  onDeleted,
}: {
  point: MunkiDistributionPoint | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}) {
  const remove = useDeleteMunkiDistributionPoint();

  async function handleDelete() {
    if (!point) return;
    await remove.mutateAsync(point.id);
    onOpenChange(false);
    toast.success("Distribution point deleted");
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete distribution point?"
      description="Clients stop being redirected to this distribution point."
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleDelete()}
    />
  );
}
