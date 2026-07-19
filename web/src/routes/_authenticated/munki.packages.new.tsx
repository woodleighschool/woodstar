import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { MunkiPackageCreatePage } from "@/pages/munki/packages/create";

const searchSchema = z.object({
  software_id: z.coerce.number().int().positive().optional(),
});

export const Route = createFileRoute("/_authenticated/munki/packages/new")({
  staticData: { breadcrumb: "New" },
  validateSearch: (search) => searchSchema.parse(search),
  component: MunkiPackageCreatePage,
});
