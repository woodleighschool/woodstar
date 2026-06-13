import { SubmitButton } from "@/components/submit-button";
import { Button } from "@/components/ui/button";
import { FieldError } from "@/components/ui/field";
import { cn } from "@/lib/utils";

export function FormActions({
  pending,
  disabled,
  error,
  submitLabel = "Save",
  onCancel,
  className,
}: {
  pending: boolean;
  disabled?: boolean;
  error?: string;
  submitLabel?: string;
  onCancel?: () => void;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col gap-2 border-t pt-4", className)}>
      <div className="flex items-center gap-2">
        <SubmitButton pending={pending} disabled={disabled} size="sm">
          {submitLabel}
        </SubmitButton>
        {onCancel ? (
          <Button type="button" variant="outline" size="sm" onClick={onCancel}>
            Cancel
          </Button>
        ) : null}
      </div>
      {error ? <FieldError>{error}</FieldError> : null}
    </div>
  );
}
