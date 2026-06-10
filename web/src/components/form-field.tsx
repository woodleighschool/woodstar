import type { AnyFieldApi } from "@tanstack/react-form";
import type { ReactNode } from "react";

import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { firstErrorMessage } from "@/lib/form-validation";

interface FieldControl {
  id: string | undefined;
  "aria-invalid": true | undefined;
}

// Wraps one @tanstack/react-form field with the Field/FieldLabel/FieldError chrome
// and the invalid wiring. The render-prop hands the control its id + aria-invalid;
// the caller binds value/onChange/onBlur off `field` so this fits any control.
export function FormField({
  field,
  label,
  htmlFor,
  required,
  description,
  className,
  children,
}: {
  field: AnyFieldApi;
  label?: ReactNode;
  htmlFor?: string;
  required?: boolean;
  description?: ReactNode;
  className?: string;
  children: (control: FieldControl) => ReactNode;
}) {
  const message = firstErrorMessage(field.state.meta.errors);
  const invalid = message ? true : undefined;
  return (
    <Field data-invalid={invalid} className={className}>
      {label ? (
        <FieldLabel htmlFor={htmlFor} required={required}>
          {label}
        </FieldLabel>
      ) : null}
      {children({ id: htmlFor, "aria-invalid": invalid })}
      {description ? <FieldDescription>{description}</FieldDescription> : null}
      {message ? <FieldError>{message}</FieldError> : null}
    </Field>
  );
}
