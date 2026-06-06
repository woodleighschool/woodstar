import { z } from "zod";

import type { MunkiAssignment, MunkiAssignmentMutation } from "@/hooks/munki/assignments";

import {
  MUNKI_ASSIGNMENT_ACTION_VALUES,
  MUNKI_PACKAGE_SELECTION_VALUES,
  type MunkiAssignmentAction,
  type MunkiPackageSelection,
} from "./shared";

export const munkiAssignmentFormSchema = z
  .object({
    priority: z.number().int("Priority must be a whole number.").positive("Priority starts at 1."),
    label_id: z
      .number()
      .int("Label selection is invalid.")
      .positive("Pick a label.")
      .nullable()
      .refine((value) => value !== null, "Pick a label."),
    package_selection: z.enum(MUNKI_PACKAGE_SELECTION_VALUES),
    pinned_package_id: z.string().trim(),
    action: z.enum(MUNKI_ASSIGNMENT_ACTION_VALUES),
    optional_install: z.boolean(),
    featured_item: z.boolean(),
  })
  .superRefine((value, ctx) => {
    if (value.package_selection === "specific_package" && !Number(value.pinned_package_id)) {
      ctx.addIssue({ code: "custom", message: "Package is required.", path: ["pinned_package_id"] });
    }
    if (value.featured_item && !value.optional_install) {
      ctx.addIssue({
        code: "custom",
        message: "Featured Items must also be Optional Installs.",
        path: ["featured_item"],
      });
    }
    if (value.action === "remove" && (value.optional_install || value.featured_item)) {
      ctx.addIssue({
        code: "custom",
        message: "Managed Uninstalls cannot also be Optional Installs or Featured Items.",
        path: ["optional_install"],
      });
    }
  });

export interface MunkiAssignmentFormState {
  priority: number;
  label_id: number | null;
  package_selection: MunkiPackageSelection;
  pinned_package_id: string;
  action: MunkiAssignmentAction;
  optional_install: boolean;
  featured_item: boolean;
}

export function emptyMunkiAssignmentForm(position = 1): MunkiAssignmentFormState {
  return {
    priority: position,
    label_id: null,
    package_selection: "latest_eligible",
    pinned_package_id: "",
    action: "install",
    optional_install: false,
    featured_item: false,
  };
}

export function munkiAssignmentFormFromAssignment(assignment: MunkiAssignment): MunkiAssignmentFormState {
  return {
    priority: assignment.priority,
    label_id: assignment.label_id,
    package_selection: assignment.package_selection,
    pinned_package_id: assignment.pinned_package_id ? String(assignment.pinned_package_id) : "",
    action: assignment.action,
    optional_install: assignment.optional_install,
    featured_item: assignment.featured_item,
  };
}

export function munkiAssignmentMutation(softwareId: number, form: MunkiAssignmentFormState): MunkiAssignmentMutation {
  const body: MunkiAssignmentMutation = {
    software_id: softwareId,
    priority: form.priority,
    label_id: form.label_id ?? 0,
    action: form.action,
    optional_install: form.optional_install,
    featured_item: form.featured_item,
    package_selection: form.package_selection,
    pinned_package_id: form.package_selection === "specific_package" ? Number(form.pinned_package_id) : undefined,
  };
  return body;
}
