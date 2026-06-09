import { z } from "zod";

import type { SoftwareInclude } from "@/lib/api";

import { MUNKI_PACKAGE_STRATEGY_VALUES, MUNKI_SOFTWARE_STATE_VALUES } from "@/lib/munki-software";

export const LATEST_PACKAGE_VALUE = "latest";

const packageSelectorSchema = z.object({
  strategy: z.enum(MUNKI_PACKAGE_STRATEGY_VALUES),
  package_id: z.number().int("Package selection is invalid.").positive("Package is required.").optional(),
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
    state: z.enum(MUNKI_SOFTWARE_STATE_VALUES),
    featured: z.boolean(),
  })
  .superRefine(validateTarget);

export interface MunkiSoftwareTargetMutation {
  priority: number;
  label_id: number | null;
  package: SoftwareInclude["package"];
  state: SoftwareInclude["state"];
  featured: boolean;
}

export function munkiSoftwareInclude(target: MunkiSoftwareTargetMutation): SoftwareInclude {
  return {
    label_id: target.label_id ?? 0,
    package: target.package,
    state: target.state,
    featured: target.featured,
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
    ctx.addIssue({ code: "custom", message: "Package is required.", path: ["package"] });
  }
  if (value.featured && value.state !== "optional_install") {
    ctx.addIssue({
      code: "custom",
      message: "Featured Items require Optional Installs.",
      path: ["featured"],
    });
  }
}
