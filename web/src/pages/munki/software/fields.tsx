import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import { FormField } from "@/components/form-field";
import { FreeTextCombobox } from "@/components/free-text-combobox";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { MunkiInclude, MunkiSoftwareDetail } from "@/lib/api";
import { requiredString } from "@/lib/form-validation";

import { MUNKI_PACKAGE_STRATEGY_VALUES, MUNKI_SOFTWARE_ACTION_VALUES } from "./munki-software";

const munkiName = requiredString("Name")
  .refine((value) => !value.includes("/"), "Name cannot contain a slash")
  .refine((value) => !/-\d[^-]*$/.test(value), "Name cannot end with a version suffix");

export const munkiSoftwareSchema = z.object({
  name: munkiName,
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
  icon_file: File | null;
  icon_object_id: number | null;
  icon_url: string;
  targets: {
    include: MunkiSoftwareTargetMutation[];
    exclude: { label_id: number }[];
  };
}

export function emptyMunkiSoftwareForm(): MunkiSoftwareFormState {
  return {
    name: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
    icon_file: null,
    icon_object_id: null,
    icon_url: "",
    targets: { include: [], exclude: [] },
  };
}

export function munkiSoftwareFormFromSoftware(title: MunkiSoftwareDetail): MunkiSoftwareFormState {
  return {
    name: title.name,
    display_name: title.display_name ?? "",
    description: title.description,
    category: title.category,
    developer: title.developer,
    icon_file: null,
    icon_object_id: title.icon_object_id ?? null,
    icon_url: title.icon_url ?? "",
    targets: {
      include: title.targets.include.map((include, index) => ({
        ...include,
        id: index + 1,
        priority: index + 1,
      })),
      exclude: title.targets.exclude,
    },
  };
}

export function useMunkiSoftwareForm(
  initial: MunkiSoftwareFormState,
  onSubmit: (value: MunkiSoftwareFormState) => Promise<number | undefined>,
  onSuccess?: (id: number) => void,
) {
  return useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: munkiSoftwareFormSchema() },
    onSubmit: async ({ value, formApi }) => {
      const id = await onSubmit(value);
      if (id === undefined) return;
      formApi.reset(value);
      onSuccess?.(id);
    },
  });
}

export type MunkiSoftwareForm = ReturnType<typeof useMunkiSoftwareForm>;

export function MunkiSoftwareOptionsFields({
  form,
  nameReadOnly = false,
  categoryOptions,
  developerOptions,
}: {
  form: MunkiSoftwareForm;
  nameReadOnly?: boolean;
  categoryOptions: string[];
  developerOptions: string[];
}) {
  return (
    <FieldGroup className="max-w-3xl">
      <form.Subscribe
        selector={(state) =>
          [state.values.icon_url, state.values.icon_file, state.values.icon_object_id] as const
        }
      >
        {([iconUrl, iconFile, iconObjectID]) => (
          <EditableMunkiIcon
            title="software icon"
            iconUrl={iconUrl || undefined}
            file={iconFile}
            clearable={!!iconFile || iconObjectID !== null}
            onFileChange={(file) => {
              form.setFieldValue("icon_file", file);
              form.setFieldValue("icon_object_id", null);
              if (file) form.setFieldValue("icon_url", "");
            }}
            onPickExisting={(object) => {
              form.setFieldValue("icon_file", null);
              form.setFieldValue("icon_object_id", object.id);
              form.setFieldValue("icon_url", object.url);
            }}
            onClear={() => {
              form.setFieldValue("icon_file", null);
              form.setFieldValue("icon_object_id", null);
              form.setFieldValue("icon_url", "");
            }}
          />
        )}
      </form.Subscribe>
      <div className="grid gap-4 md:grid-cols-2">
        <form.Field name="name">
          {(field) => (
            <FormField field={field} label="Name" htmlFor="munki-software-name" required>
              {(control) => (
                <Input
                  {...control}
                  name={field.name}
                  value={field.state.value}
                  readOnly={nameReadOnly}
                  onBlur={field.handleBlur}
                  onChange={(event) => field.handleChange(event.target.value)}
                />
              )}
            </FormField>
          )}
        </form.Field>
        <form.Field name="display_name">
          {(field) => (
            <FormField field={field} label="Display Name" htmlFor="munki-software-display-name">
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
    id: z.number().int().positive(),
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

export function munkiSoftwareFormSchema() {
  return munkiSoftwareSchema.extend({
    icon_file: z.custom<File | null>((value) => value === null || value instanceof File),
    icon_object_id: z.number().int().positive().nullable(),
    icon_url: z.string(),
    targets: z.object({
      include: z.array(munkiSoftwareTargetSchema),
      exclude: z.array(
        z.object({
          label_id: z.number().int("Label selection is invalid.").positive("Pick a label."),
        }),
      ),
    }),
  });
}

export interface MunkiSoftwareTargetMutation {
  id: number;
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
