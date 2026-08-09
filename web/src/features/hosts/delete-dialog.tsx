import { ConfirmDialog } from "@components/confirm-dialog";
import { toast } from "@components/ui/toast";
import { useDeleteHost } from "@features/hosts/queries";
import type { Host } from "@lib/api";

export function HostDeleteDialog({
  host,
  open,
  onOpenChange,
  onDeleted,
}: {
  host: Host | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}) {
  const remove = useDeleteHost();

  async function handleDelete() {
    if (!host) return;
    await remove.mutateAsync(host.id);
    onOpenChange(false);
    toast.add({ title: "Host deleted", type: "success" });
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) remove.reset();
        onOpenChange(next);
      }}
      title="Delete Host"
      description={
        host
          ? `${host.display_name} and its collected state will be permanently deleted. The agent can re-enroll with a valid Orbit secret.`
          : "This host and its collected state will be permanently deleted."
      }
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleDelete()}
    />
  );
}
