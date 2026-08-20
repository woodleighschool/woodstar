import { ConfirmDialog } from "@components/confirm-dialog";
import { toast } from "@components/ui/toast";
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
    toast.add({ title: "Distribution Point Deleted", type: "success" });
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete Distribution Point?"
      description="Clients stop being redirected to this distribution point."
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleDelete()}
    />
  );
}
