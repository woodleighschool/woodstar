import type { VariantProps } from "class-variance-authority";

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
import { buttonVariants } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

export interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  confirmLabel: string;
  variant?: VariantProps<typeof buttonVariants>["variant"];
  pending?: boolean;
  onConfirm: () => void;
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  variant,
  pending = false,
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          {description ? <AlertDialogDescription>{description}</AlertDialogDescription> : null}
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel variant="ghost" size="sm" disabled={pending}>
            Cancel
          </AlertDialogCancel>
          <Pending
            isPending={pending}
            render={
              <AlertDialogAction
                variant={variant}
                size="sm"
                onClick={(event) => {
                  event.preventDefault();
                  onConfirm();
                }}
              />
            }
          >
            {pending ? <Spinner data-icon="inline-start" /> : null}
            {confirmLabel}
          </Pending>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
