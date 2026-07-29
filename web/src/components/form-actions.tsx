import type { AnyFormApi } from "@tanstack/react-form";
import { useSelector } from "@tanstack/react-store";
import type { ReactNode } from "react";

import { AsyncButton } from "@components/async-button";
import { Pending } from "@components/pending";
import { Button } from "@components/ui/button";
import { Field } from "@components/ui/field";
import { cn } from "@lib/utils";
// Invalid forms stay submittable so a submit attempt can reveal every field
// error. Pending state is reserved for an active submission.
export function FormActions({
  form,
  submitLabel,
  onSubmit,
  onCancel,
  canCancelWhileSubmitting = false,
  className,
  children,
}: {
  form: AnyFormApi;
  submitLabel: string;
  onSubmit?: () => Promise<unknown> | void;
  onCancel?: () => void;
  canCancelWhileSubmitting?: boolean;
  className?: string;
  children?: ReactNode;
}) {
  const isSubmitting = useSelector(form.store, (state) => state.isSubmitting);
  // Submit explicitly so unrelated controls such as table searches and pickers
  // are never enrolled in a page-wide native form.
  const submit = () => {
    if (onSubmit) {
      void onSubmit();
      return;
    }
    void form.handleSubmit();
  };
  return (
    <Field orientation="horizontal" className={cn("justify-start", className)}>
      <AsyncButton isPending={isSubmitting} type="button" size="sm" onClick={submit}>
        {submitLabel}
      </AsyncButton>
      {onCancel ? (
        <Pending
          isPending={isSubmitting && !canCancelWhileSubmitting}
          render={<Button type="button" variant="outline" size="sm" onClick={onCancel} />}
        >
          Cancel
        </Pending>
      ) : null}
      {children}
    </Field>
  );
}
