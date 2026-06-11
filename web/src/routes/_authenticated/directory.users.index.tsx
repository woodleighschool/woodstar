import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { UserListPage } from "@/pages/users/list";

// Table state (q, page, per_page, sort, role + source facets) is nuqs-owned; the
// route only validates the semantic group_id deep-link and stays loose so the
// nuqs keys survive validation on a bookmarked load.
const searchSchema = z.looseObject({
  group_id: z.coerce.number().int().positive().optional(),
});

export const Route = createFileRoute("/_authenticated/directory/users/")({
  validateSearch: (search) => searchSchema.parse(search),
  component: UserListPage,
});
