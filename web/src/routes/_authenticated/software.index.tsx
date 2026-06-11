import { createFileRoute } from "@tanstack/react-router";

import { SoftwareListPage } from "@/pages/software/list";

// Pure list route: q, page, per_page, sort, and the source facet are nuqs-owned.
export const Route = createFileRoute("/_authenticated/software/")({
  component: SoftwareListPage,
});
