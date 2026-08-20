import { ConfirmDialog } from "@components/confirm-dialog";
import { toast } from "@components/ui/toast";
import type { SantaConfiguration } from "@lib/api";

import { useDeleteSantaConfiguration } from "./queries";

export function ConfigurationDeleteDialog({
  configuration,
  open,
  onOpenChange,
  onDeleted,
}: {
  configuration: SantaConfiguration | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}) {
  const remove = useDeleteSantaConfiguration();

  async function handleDelete() {
    if (!configuration) return;
    await remove.mutateAsync(configuration.id);
    onOpenChange(false);
    toast.add({ title: "Configuration Deleted", type: "success" });
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete Configuration?"
      description={
        configuration
          ? `${configuration.name} will stop applying to targeted Santa clients.`
          : "This configuration will stop applying."
      }
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleDelete()}
    />
  );
}
