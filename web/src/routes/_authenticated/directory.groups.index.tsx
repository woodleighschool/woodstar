import { createFileRoute } from "@tanstack/react-router";

import { GroupListPage } from "@/pages/groups/list";

// Pure list route: q, page, per_page, and sort are nuqs-owned, so there are no
// semantic search params to validate.
export const Route = createFileRoute("/_authenticated/directory/groups/")({
  component: GroupListPage,
});
