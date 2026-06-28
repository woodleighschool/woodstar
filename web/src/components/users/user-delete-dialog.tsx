import { toast } from "sonner";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { useDeleteUser } from "@/hooks/use-users";
import type { User } from "@/lib/api";
import { nonEmpty } from "@/lib/utils";

export interface UserDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: User | null;
  onDeleted?: () => void;
}

export function UserDeleteDialog({ open, onOpenChange, user, onDeleted }: UserDeleteDialogProps) {
  const remove = useDeleteUser();
  const isLocal = user?.source === "local";
  const action = "Delete User";
  const name = nonEmpty(user?.name) ?? nonEmpty(user?.email) ?? "";
  const userDescription = user?.name ? `${name} (${user.email})` : name;
  const description = isLocal
    ? "This permanently deletes the local user. Their next request will sign them out automatically."
    : "This removes the user from Woodstar's current directory view. Directory sync can restore the user if the source still contains it.";

  async function handleConfirm() {
    if (!user) return;
    await remove.mutateAsync(user.id);
    onOpenChange(false);
    toast.success("User deleted");
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) remove.reset();
        onOpenChange(next);
      }}
      title={action}
      description={userDescription ? `${description} Delete ${userDescription}?` : description}
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleConfirm()}
    />
  );
}
