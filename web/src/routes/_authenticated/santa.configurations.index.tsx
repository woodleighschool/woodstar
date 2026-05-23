import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { SantaConfigurationsPage } from "@/pages/santa/configurations";

const searchSchema = z.object({
  q: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/santa/configurations/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SantaConfigurationsPage,
});
