import { ConfirmDialog } from "@components/confirm-dialog";
import { toast } from "@components/ui/toast";
import type { MunkiPackage } from "@lib/api";

import { useDeleteMunkiPackage } from "./queries";

export function MunkiPackageDeleteDialog({
  pkg,
  open,
  onOpenChange,
  onDeleted,
}: {
  pkg: MunkiPackage | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}) {
  const remove = useDeleteMunkiPackage();

  async function handleDelete() {
    if (!pkg) return;
    await remove.mutateAsync(pkg.id);
    onOpenChange(false);
    toast.add({ title: "Package Deleted", type: "success" });
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete Package?"
      description={
        pkg
          ? `${pkg.software.name} ${pkg.version} will be deleted. Referenced packages cannot be deleted.`
          : "Referenced packages cannot be deleted."
      }
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleDelete()}
    />
  );
}
