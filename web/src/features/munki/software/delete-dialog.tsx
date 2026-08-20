import { ConfirmDialog } from "@components/confirm-dialog";
import { toast } from "@components/ui/toast";
import type { MunkiSoftware } from "@lib/api";

import { useDeleteMunkiSoftware } from "./queries";

export function MunkiSoftwareDeleteDialog({
  software,
  open,
  onOpenChange,
  onDeleted,
}: {
  software: MunkiSoftware | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}) {
  const remove = useDeleteMunkiSoftware();

  async function handleDelete() {
    if (!software) return;
    await remove.mutateAsync(software.id);
    onOpenChange(false);
    toast.add({ title: "Software Deleted", type: "success" });
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete Software?"
      description={
        software
          ? `${software.name} and its packages and targeting will be deleted.`
          : "This software and its packages and targeting will be deleted."
      }
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleDelete()}
    />
  );
}
