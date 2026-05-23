import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useDeleteUser, type User } from "@/hooks/use-users";
import { nonEmpty } from "@/lib/utils";

export interface UserDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: User | null;
  onDeleted?: () => void;
}

export function UserDeleteDialog({ open, onOpenChange, user, onDeleted }: UserDeleteDialogProps) {
  const remove = useDeleteUser();

  async function handleConfirm() {
    if (!user) return;
    await remove.mutateAsync(user.id);
    onOpenChange(false);
    onDeleted?.();
  }

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) remove.reset();
        onOpenChange(next);
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete user</AlertDialogTitle>
          <AlertDialogDescription>
            This permanently deletes the user. Their next request will sign them out automatically.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <p className="text-sm">
          Delete <span className="font-medium">{nonEmpty(user?.name) ?? nonEmpty(user?.email) ?? ""}</span>
          {user?.name ? <span className="text-muted-foreground"> ({user.email})</span> : null}?
        </p>

        {remove.error ? <p className="text-sm text-destructive">{remove.error.message}</p> : null}

        <AlertDialogFooter>
          <AlertDialogCancel variant="ghost" size="sm" disabled={remove.isPending}>
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            size="sm"
            disabled={remove.isPending}
            onClick={(event) => {
              event.preventDefault();
              void handleConfirm();
            }}
          >
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
