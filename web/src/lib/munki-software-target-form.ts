import { z } from "zod";

import type { SoftwareInclude } from "@/lib/api";

import {
  MUNKI_PACKAGE_STRATEGY_VALUES,
  MUNKI_SOFTWARE_STATE_VALUES,
  type MunkiPackageStrategy,
  type MunkiSoftwareState,
} from "@/lib/munki-software";

const munkiSoftwareTargetDetailsFields = {
  strategy: z.enum(MUNKI_PACKAGE_STRATEGY_VALUES),
  package_id: z.string().trim(),
  state: z.enum(MUNKI_SOFTWARE_STATE_VALUES),
  featured: z.boolean(),
};

export const munkiSoftwareTargetDetailsFormSchema = z
  .object(munkiSoftwareTargetDetailsFields)
  .superRefine(validateDetails);

export const munkiSoftwareTargetFormSchema = z
  .object({
    priority: z.number().int("Priority must be a whole number.").positive("Priority starts at 1."),
    label_id: z
      .number()
      .int("Label selection is invalid.")
      .positive("Pick a label.")
      .nullable()
      .refine((value) => value !== null, "Pick a label."),
    ...munkiSoftwareTargetDetailsFields,
  })
  .superRefine(validateDetails);

export interface MunkiSoftwareTargetFormState {
  priority: number;
  label_id: number | null;
  strategy: MunkiPackageStrategy;
  package_id: string;
  state: MunkiSoftwareState;
  featured: boolean;
}

type MunkiSoftwareTargetDetailsFormState = Pick<
  MunkiSoftwareTargetFormState,
  "strategy" | "package_id" | "state" | "featured"
>;

export function emptyMunkiSoftwareTargetForm(position = 1): MunkiSoftwareTargetFormState {
  return {
    priority: position,
    label_id: null,
    strategy: "latest",
    package_id: "",
    state: "managed_install",
    featured: false,
  };
}

export function munkiSoftwareInclude(form: MunkiSoftwareTargetFormState): SoftwareInclude {
  return {
    label_id: form.label_id ?? 0,
    package: {
      strategy: form.strategy,
      package_id: form.strategy === "specific" ? Number(form.package_id) : undefined,
    },
    state: form.state,
    featured: form.featured,
  };
}

function validateDetails(value: MunkiSoftwareTargetDetailsFormState, ctx: z.RefinementCtx) {
  if (value.strategy === "specific" && !Number(value.package_id)) {
    ctx.addIssue({ code: "custom", message: "Package is required.", path: ["package_id"] });
  }
  if (value.featured && value.state !== "optional_install") {
    ctx.addIssue({
      code: "custom",
      message: "Featured Items require Optional Installs.",
      path: ["featured"],
    });
  }
}
