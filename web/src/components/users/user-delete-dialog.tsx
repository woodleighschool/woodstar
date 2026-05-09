import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useDeleteUser, type User } from "@/hooks/use-users";
import { nonEmpty } from "@/lib/utils";

export interface UserDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: User | null;
}

export function UserDeleteDialog({ open, onOpenChange, user }: UserDeleteDialogProps) {
  const remove = useDeleteUser();

  async function handleConfirm() {
    if (!user) return;
    await remove.mutateAsync(user.id);
    onOpenChange(false);
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) remove.reset();
        onOpenChange(next);
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Delete user</DialogTitle>
          <DialogDescription>
            This soft-deletes the user and revokes all of their sessions. They will be signed out immediately.
          </DialogDescription>
        </DialogHeader>

        <p className="text-sm">
          Delete <span className="font-medium">{nonEmpty(user?.name) ?? nonEmpty(user?.email) ?? ""}</span>
          {user?.name ? <span className="text-muted-foreground"> ({user.email})</span> : null}?
        </p>

        {remove.error ? <p className="text-sm text-destructive">{remove.error.message}</p> : null}

        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="ghost" size="sm" disabled={remove.isPending}>
              Cancel
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            disabled={remove.isPending}
            onClick={() => void handleConfirm()}
          >
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
