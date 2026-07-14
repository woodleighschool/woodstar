import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import { FormField } from "@/components/form-field";
import { FreeTextCombobox } from "@/components/free-text-combobox";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { FieldDescription, FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { MunkiInclude, MunkiSoftwareDetail } from "@/lib/api";
import { requiredString } from "@/lib/form-validation";

import { MUNKI_PACKAGE_STRATEGY_VALUES, MUNKI_SOFTWARE_ACTION_VALUES } from "./munki-software";

export const munkiSoftwareSchema = z.object({
  name: requiredString("Munki name"),
  display_name: z.string().trim(),
  description: z.string().trim(),
  category: z.string().trim(),
  developer: z.string().trim(),
});

export interface MunkiSoftwareFormState {
  name: string;
  display_name: string;
  description: string;
  category: string;
  developer: string;
}

export function emptyMunkiSoftwareForm(): MunkiSoftwareFormState {
  return { name: "", display_name: "", description: "", category: "", developer: "" };
}

export function munkiSoftwareFormFromSoftware(title: MunkiSoftwareDetail): MunkiSoftwareFormState {
  return {
    name: title.name,
    display_name: title.display_name,
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
    validationLogic: revalidateLogic(),
    validators: { onDynamic: munkiSoftwareSchema },
    onSubmit: ({ value }) => onSubmit(value),
  });
}

export type MunkiSoftwareForm = ReturnType<typeof useMunkiSoftwareForm>;

export type MunkiSoftwareIconProps = {
  iconUrl?: string;
  file: File | null;
  clearable: boolean;
  onFileChange: (file: File | null) => void;
  onPickExisting: (object: { id: number; url: string }) => void;
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
        The Munki name identifies the software in manifests and package relationships. The display
        name is shown in Managed Software Center.
      </FieldDescription>
      <div className="flex items-start gap-4">
        <EditableMunkiIcon
          title="software icon"
          iconUrl={icon.iconUrl}
          file={icon.file}
          clearable={icon.clearable}
          onFileChange={icon.onFileChange}
          onPickExisting={icon.onPickExisting}
          onClear={icon.onClear}
        />
        <div className="grid min-w-0 flex-1 gap-4 md:grid-cols-2">
          <form.Field name="name">
            {(field) => (
              <FormField
                field={field}
                label="Munki name"
                htmlFor="munki-software-name"
                description="Canonical name used by Munki."
                required
              >
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
          <form.Field name="display_name">
            {(field) => (
              <FormField
                field={field}
                label="Display name"
                htmlFor="munki-software-display-name"
                description="Falls back to the Munki name when empty."
              >
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
                  items={categoryOptions}
                  itemToStringValue={(option) => option}
                  freeTextItem={freeTextOption}
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
                  items={developerOptions}
                  itemToStringValue={(option) => option}
                  freeTextItem={freeTextOption}
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

function freeTextOption(value: string) {
  return value;
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
  package: MunkiInclude["package"];
  actions: MunkiInclude["actions"];
}

export function munkiSoftwareInclude(target: MunkiSoftwareTargetMutation): MunkiInclude {
  return {
    label_id: target.label_id ?? 0,
    package: target.package,
    actions: target.actions,
  };
}

export function targetPackageValue(selector: MunkiInclude["package"]) {
  if (selector.strategy === "specific" && selector.package_id) {
    return String(selector.package_id);
  }
  return LATEST_PACKAGE_VALUE;
}

export function targetPackageFromValue(value: string): MunkiInclude["package"] {
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
