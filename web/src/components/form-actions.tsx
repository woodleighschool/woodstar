import type { AnyFormApi } from "@tanstack/react-form";
import { useStore } from "@tanstack/react-form";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

// Submit/cancel footer for every form. Submit is gated on the form's own state:
// disabled while submitting or invalid (canSubmit), and while nothing has changed
// from the initial values (isDefaultValue). Forms that keep part of their state
// outside the form (uploads, separate editors) pass requireDirty={false} so the
// dirty check doesn't suppress those changes. Outcomes are reported by toast, so
// there is no spinner or inline error here.
export function FormActions({
  form,
  submitLabel = "Save",
  onCancel,
  className,
  requireDirty = true,
}: {
  form: AnyFormApi;
  submitLabel?: string;
  onCancel?: () => void;
  className?: string;
  requireDirty?: boolean;
}) {
  const canSubmit = useStore(form.store, (state) => state.canSubmit);
  const isDefaultValue = useStore(form.store, (state) => state.isDefaultValue);
  return (
    <div className={cn("flex items-center gap-2 border-t pt-4", className)}>
      <Button type="submit" size="sm" disabled={!canSubmit || (requireDirty && isDefaultValue)}>
        {submitLabel}
      </Button>
      {onCancel ? (
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          Cancel
        </Button>
      ) : null}
    </div>
  );
}
