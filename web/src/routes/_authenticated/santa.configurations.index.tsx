import { createFileRoute } from "@tanstack/react-router";

import { ConfigurationListPage } from "@/pages/santa/configurations/list";

// Pure list route: q, page, per_page, and sort are nuqs-owned.
export const Route = createFileRoute("/_authenticated/santa/configurations/")({
  component: ConfigurationListPage,
});
