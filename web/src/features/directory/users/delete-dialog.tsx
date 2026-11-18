import { ConfirmDialog } from "@components/confirm-dialog";
import { toast } from "@components/ui/toast";
import { useDeleteUser } from "@features/directory/users/queries";
import type { User } from "@lib/api";
import { nonEmpty } from "@lib/utils";

export interface UserDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: User | null;
  onDeleted?: () => void;
}

export function UserDeleteDialog({ open, onOpenChange, user, onDeleted }: UserDeleteDialogProps) {
  const remove = useDeleteUser();
  const name = nonEmpty(user?.name);
  const label = name ? `${name} (${user?.email})` : (user?.email ?? "this user");
  const description =
    user?.source === "local"
      ? `Permanently delete ${label}?`
      : `Delete ${label}? Sync will restore this user if they are still in the directory.`;

  function handleConfirm() {
    if (!user) return;
    remove.mutate(user.id, {
      onSuccess: () => {
        onOpenChange(false);
        toast.add({ title: "User Deleted", type: "success" });
        onDeleted?.();
      },
    });
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) remove.reset();
        onOpenChange(next);
      }}
      title="Delete User"
      description={description}
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={handleConfirm}
    />
  );
}
