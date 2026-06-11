import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { SantaEventListPage } from "@/pages/santa/events/list";

// Table state (q, page, per_page, sort, decision facet) is nuqs-owned; the route
// validates the host_id/user deep-link context and stays loose so nuqs keys
// survive validation on a bookmarked load.
const searchSchema = z.looseObject({
  host_id: z.coerce.number().int().positive().optional(),
  user: z.string().optional(),
});

export const Route = createFileRoute("/_authenticated/santa/events/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: SantaEventListPage,
});
