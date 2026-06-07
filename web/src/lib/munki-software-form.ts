import { z } from "zod";

import type { MunkiSoftwareDetail } from "@/hooks/munki/software";
import { requiredString } from "@/lib/form-validation";

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
  return {
    name: "",
    description: "",
    category: "",
    developer: "",
  };
}

export function munkiSoftwareFormFromSoftware(title: MunkiSoftwareDetail): MunkiSoftwareFormState {
  return {
    name: title.name,
    description: title.description,
    category: title.category,
    developer: title.developer,
  };
}
