import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { HostListPage } from "@/pages/hosts/list";

// Table state (q, page, per_page, sort, status facet) is nuqs-owned; the route
// only validates the semantic deep-link params and stays loose so the nuqs keys
// survive validation on a bookmarked load.
const searchSchema = z.looseObject({
  label_id: z.string().optional(),
  software_title_id: z.coerce.number().int().positive().optional(),
  software_id: z.coerce.number().int().positive().optional(),
  check_id: z.coerce.number().int().positive().optional(),
  check_response: z.enum(["pass", "fail"]).optional(),
});

export const Route = createFileRoute("/_authenticated/hosts/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: HostListPage,
});
