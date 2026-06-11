import { createFileRoute } from "@tanstack/react-router";

import { LabelListPage } from "@/pages/labels/list";

// Pure list route: q, page, per_page, sort, and the membership facet are all
// nuqs-owned, so there are no semantic search params to validate.
export const Route = createFileRoute("/_authenticated/labels/")({
  component: LabelListPage,
});
