import { Link } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { type ReactNode } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldContent, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";

export function TextField({
  id,
  label,
  required,
  value,
  error,
  placeholder,
  description,
  onChange,
}: {
  id: string;
  label: string;
  required?: boolean;
  value: string;
  error?: string;
  placeholder?: string;
  description?: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field data-invalid={error ? true : undefined}>
      <FieldLabel htmlFor={id} required={required}>
        {label}
      </FieldLabel>
      <Input id={id} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />
      {description ? <FieldDescription>{description}</FieldDescription> : null}
      {error ? <FieldError>{error}</FieldError> : null}
    </Field>
  );
}

export function StringListField({
  id,
  label,
  description,
  value,
  onChange,
}: {
  id: string;
  label: string;
  description: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input id={id} value={value} placeholder="ItemA, ItemB" onChange={(event) => onChange(event.target.value)} />
      <FieldDescription>{description}</FieldDescription>
    </Field>
  );
}

export function DatalistTextField({
  id,
  label,
  value,
  options,
  placeholder,
  onChange,
}: {
  id: string;
  label: string;
  value: string;
  options: string[];
  placeholder?: string;
  onChange: (value: string) => void;
}) {
  const listID = `${id}-options`;
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input
        id={id}
        list={options.length > 0 ? listID : undefined}
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
      {options.length > 0 ? (
        <datalist id={listID}>
          {options.map((option) => (
            <option key={option} value={option} />
          ))}
        </datalist>
      ) : null}
    </Field>
  );
}

export function SelectField<T extends string>({
  id,
  label,
  value,
  options,
  description,
  onChange,
}: {
  id: string;
  label: string;
  value: T;
  options: Array<{ value: T; label: string; description?: string }>;
  description?: string;
  onChange: (value: T) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Select value={value} onValueChange={(next) => onChange(next as T)}>
        <SelectTrigger id={id} className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      {description ? <FieldDescription>{description}</FieldDescription> : null}
    </Field>
  );
}

export function FileField({
  id,
  label,
  description,
  accept,
  icon,
  file,
  onChange,
}: {
  id: string;
  label: string;
  description: string;
  accept?: string;
  icon: ReactNode;
  file: File | null;
  onChange: (file: File | null) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <div className="flex items-center gap-3">
        <div className="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-md border">
          {icon}
        </div>
        <Input id={id} type="file" accept={accept} onChange={(event) => onChange(event.target.files?.[0] ?? null)} />
      </div>
      <FieldDescription>{file ? file.name : description}</FieldDescription>
    </Field>
  );
}

export function TextAreaField({
  id,
  label,
  value,
  placeholder,
  onChange,
}: {
  id: string;
  label: string;
  value: string;
  placeholder?: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Textarea id={id} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />
    </Field>
  );
}

export function CheckboxField({
  id,
  label,
  description,
  checked,
  disabled,
  onChange,
}: {
  id: string;
  label: string;
  description?: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <Field orientation="horizontal" className={disabled ? "opacity-60" : undefined}>
      <Checkbox id={id} checked={checked} disabled={disabled} onCheckedChange={(value) => onChange(value === true)} />
      <FieldContent>
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
        {description ? <FieldDescription>{description}</FieldDescription> : null}
      </FieldContent>
    </Field>
  );
}

export function FormActions({
  pending,
  cancelTo,
  cancelParams,
}: {
  pending: boolean;
  cancelTo: string;
  cancelParams?: Record<string, string>;
}) {
  return (
    <div className="flex items-center gap-2">
      <Button type="submit" size="sm" disabled={pending}>
        {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
        Save
      </Button>
      <Button asChild type="button" variant="outline" size="sm">
        <Link to={cancelTo} params={cancelParams}>
          Cancel
        </Link>
      </Button>
    </div>
  );
}

export function MutationError({ title, message }: { title: string; message?: string }) {
  if (!message) return null;
  return (
    <Alert variant="destructive">
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}
