import { useForm } from "@tanstack/react-form";
import type { ReactNode } from "react";
import { z } from "zod";

import { FormField } from "@/components/form-field";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { FreeTextCombobox } from "@/components/munki/free-text-combobox";
import { SubmitButton } from "@/components/submit-button";
import { FieldDescription, FieldError, FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { MunkiSoftwareDetail } from "@/hooks/use-munki-software";
import type { SoftwareInclude } from "@/lib/api";
import { requiredString } from "@/lib/form-validation";

import { MUNKI_PACKAGE_STRATEGY_VALUES, MUNKI_SOFTWARE_ACTION_VALUES } from "./munki-software";

export const munkiSoftwareSchema = z.object({
  name: requiredString("Name"),
  description: z.string().trim(),
  category: z.string().trim(),
  developer: z.string().trim(),
});

export interface MunkiSoftwareFormState {
  name: string;
  description: string;
  category: string;
  developer: string;
}

export function emptyMunkiSoftwareForm(): MunkiSoftwareFormState {
  return { name: "", description: "", category: "", developer: "" };
}

export function munkiSoftwareFormFromSoftware(title: MunkiSoftwareDetail): MunkiSoftwareFormState {
  return {
    name: title.name,
    description: title.description,
    category: title.category,
    developer: title.developer,
  };
}

export function useMunkiSoftwareForm(
  initial: MunkiSoftwareFormState,
  onSubmit: (value: MunkiSoftwareFormState) => Promise<void>,
) {
  return useForm({
    defaultValues: initial,
    validators: {
      onSubmit: munkiSoftwareSchema,
    },
    onSubmit: ({ value }) => onSubmit(value),
  });
}

export type MunkiSoftwareForm = ReturnType<typeof useMunkiSoftwareForm>;

export type MunkiSoftwareIconProps = {
  iconUrl?: string;
  file: File | null;
  clearable: boolean;
  onFileChange: (file: File | null) => void;
  onClear: () => void;
};

export function MunkiSoftwareOptionsFields({
  form,
  categoryOptions,
  developerOptions,
  icon,
}: {
  form: MunkiSoftwareForm;
  categoryOptions: string[];
  developerOptions: string[];
  icon: MunkiSoftwareIconProps;
}) {
  return (
    <FieldGroup className="max-w-3xl">
      <FieldDescription>
        These fields are visible to users in Managed Software Center.
      </FieldDescription>
      <div className="flex items-start gap-4">
        <EditableMunkiIcon
          title="software icon"
          iconUrl={icon.iconUrl}
          file={icon.file}
          clearable={icon.clearable}
          onFileChange={icon.onFileChange}
          onClear={icon.onClear}
        />
        <div className="min-w-0 flex-1">
          <form.Field name="name">
            {(field) => (
              <FormField field={field} label="Name" htmlFor="munki-software-name" required>
                {(control) => (
                  <Input
                    {...control}
                    name={field.name}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </FormField>
            )}
          </form.Field>
        </div>
      </div>
      <form.Field name="description">
        {(field) => (
          <FormField field={field} htmlFor="munki-software-description" label="Description">
            {(control) => (
              <Textarea
                {...control}
                name={field.name}
                value={field.state.value}
                onBlur={field.handleBlur}
                onChange={(event) => field.handleChange(event.target.value)}
              />
            )}
          </FormField>
        )}
      </form.Field>
      <div className="grid gap-4 md:grid-cols-2">
        <form.Field name="category">
          {(field) => (
            <FormField field={field} label="Category" htmlFor="munki-software-category">
              {(control) => (
                <FreeTextCombobox
                  id={control.id}
                  name={field.name}
                  value={field.state.value}
                  options={categoryOptions}
                  invalid={control["aria-invalid"]}
                  onBlur={field.handleBlur}
                  onChange={field.handleChange}
                />
              )}
            </FormField>
          )}
        </form.Field>
        <form.Field name="developer">
          {(field) => (
            <FormField field={field} label="Developer" htmlFor="munki-software-developer">
              {(control) => (
                <FreeTextCombobox
                  id={control.id}
                  name={field.name}
                  value={field.state.value}
                  options={developerOptions}
                  invalid={control["aria-invalid"]}
                  onBlur={field.handleBlur}
                  onChange={field.handleChange}
                />
              )}
            </FormField>
          )}
        </form.Field>
      </div>
    </FieldGroup>
  );
}

export function MunkiSoftwareFormActions({
  pending,
  error,
  cancel,
}: {
  pending: boolean;
  error?: string;
  cancel: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-2 border-t pt-4">
      {error ? <FieldError>{error}</FieldError> : null}
      <div className="flex items-center gap-2">
        <SubmitButton pending={pending} size="sm">
          Save
        </SubmitButton>
        {cancel}
      </div>
    </div>
  );
}

export const LATEST_PACKAGE_VALUE = "latest";

const packageSelectorSchema = z.object({
  strategy: z.enum(MUNKI_PACKAGE_STRATEGY_VALUES),
  package_id: z
    .number()
    .int("Package selection is invalid.")
    .positive("Package is required.")
    .optional(),
});

export const munkiSoftwareTargetSchema = z
  .object({
    priority: z.number().int("Priority must be a whole number.").positive("Priority starts at 1."),
    label_id: z
      .number()
      .int("Label selection is invalid.")
      .positive("Pick a label.")
      .nullable()
      .refine((value) => value !== null, "Pick a label."),
    package: packageSelectorSchema,
    actions: z.array(z.enum(MUNKI_SOFTWARE_ACTION_VALUES)).min(1, "Pick at least one action."),
  })
  .superRefine(validateTarget);

export interface MunkiSoftwareTargetMutation {
  priority: number;
  label_id: number | null;
  package: SoftwareInclude["package"];
  actions: SoftwareInclude["actions"];
}

export function munkiSoftwareInclude(target: MunkiSoftwareTargetMutation): SoftwareInclude {
  return {
    label_id: target.label_id ?? 0,
    package: target.package,
    actions: target.actions,
  };
}

export function targetPackageValue(selector: SoftwareInclude["package"]) {
  if (selector.strategy === "specific" && selector.package_id) {
    return String(selector.package_id);
  }
  return LATEST_PACKAGE_VALUE;
}

export function targetPackageFromValue(value: string): SoftwareInclude["package"] {
  if (value === LATEST_PACKAGE_VALUE) {
    return { strategy: "latest" };
  }
  return { strategy: "specific", package_id: Number(value) };
}

function validateTarget(value: MunkiSoftwareTargetMutation, ctx: z.RefinementCtx) {
  if (value.package.strategy === "latest" && value.package.package_id !== undefined) {
    ctx.addIssue({
      code: "custom",
      message: "Latest must not pin a package.",
      path: ["package"],
    });
  }
  if (value.package.strategy === "specific" && !value.package.package_id) {
    ctx.addIssue({
      code: "custom",
      message: "Package is required.",
      path: ["package"],
    });
  }
}
