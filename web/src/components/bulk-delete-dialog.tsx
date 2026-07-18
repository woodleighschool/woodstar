import { Pending } from "@/components/pending";
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
import { Spinner } from "@/components/ui/spinner";

export interface BulkDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  count: number;
  noun: string;
  pluralNoun?: string;
  description?: string;
  pending?: boolean;
  onConfirm: () => void;
}

export function BulkDeleteDialog({
  open,
  onOpenChange,
  count,
  noun,
  pluralNoun,
  description,
  pending = false,
  onConfirm,
}: BulkDeleteDialogProps) {
  const label = count === 1 ? noun : (pluralNoun ?? `${noun}s`);

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            Delete {count} Selected {label}?
          </AlertDialogTitle>
          {description ? <AlertDialogDescription>{description}</AlertDialogDescription> : null}
        </AlertDialogHeader>

        <AlertDialogFooter>
          <AlertDialogCancel variant="ghost" size="sm" disabled={pending}>
            Cancel
          </AlertDialogCancel>
          <Pending
            isPending={pending}
            disabled={count === 0}
            render={
              <AlertDialogAction
                variant="destructive"
                size="sm"
                disabled={count === 0}
                onClick={(event) => {
                  event.preventDefault();
                  onConfirm();
                }}
              />
            }
          >
            {pending ? <Spinner data-icon="inline-start" /> : null}
            Delete
          </Pending>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
