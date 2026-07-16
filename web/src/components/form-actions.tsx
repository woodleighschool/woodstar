import type { AnyFormApi } from "@tanstack/react-form";
import { useSelector } from "@tanstack/react-store";

import { Pending } from "@/components/pending";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { cn } from "@/lib/utils";

// Invalid forms stay submittable so a submit attempt can reveal every field
// error. Pending state is reserved for an active submission.
export function FormActions({
  form,
  submitLabel,
  onCancel,
  canCancelWhileSubmitting = false,
  className,
}: {
  form: AnyFormApi;
  submitLabel: string;
  onCancel?: () => void;
  canCancelWhileSubmitting?: boolean;
  className?: string;
}) {
  const isSubmitting = useSelector(form.store, (state) => state.isSubmitting);

  return (
    <Field orientation="horizontal" className={cn("justify-start", className)}>
      <Pending isPending={isSubmitting}>
        <Button type="submit" size="sm">
          {isSubmitting ? `${submitLabel}…` : submitLabel}
        </Button>
      </Pending>
      {onCancel ? (
        <Pending isPending={isSubmitting && !canCancelWhileSubmitting}>
          <Button type="button" variant="outline" size="sm" onClick={onCancel}>
            Cancel
          </Button>
        </Pending>
      ) : null}
    </Field>
  );
}
