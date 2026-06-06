import { z } from "zod";

import type { MunkiSoftwareTitleDetail } from "@/hooks/munki/software-titles";
import { requiredString } from "@/lib/form-validation";

export const softwareTitleSchema = z.object({
  name: requiredString("Name"),
  display_name: z.string().trim(),
  description: z.string().trim(),
  category: z.string().trim(),
  developer: z.string().trim(),
});

export interface SoftwareTitleFormState {
  name: string;
  display_name: string;
  description: string;
  category: string;
  developer: string;
}

export function emptySoftwareTitleForm(): SoftwareTitleFormState {
  return {
    name: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
  };
}

export function softwareTitleFormFromTitle(title: MunkiSoftwareTitleDetail): SoftwareTitleFormState {
  return {
    name: title.name,
    display_name: title.display_name,
    description: title.description,
    category: title.category,
    developer: title.developer,
  };
}
