import { createFileRoute } from "@tanstack/react-router";

import { CheckListPage } from "@/pages/osquery/checks/list";

// Pure list route: q, page, per_page, and sort are nuqs-owned.
export const Route = createFileRoute("/_authenticated/osquery/checks/")({
  component: CheckListPage,
});
