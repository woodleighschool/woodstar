import { toast } from "sonner";

import { ConfirmDialog } from "@components/confirm-dialog";
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
    toast.success("Configuration deleted");
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Delete configuration?"
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
