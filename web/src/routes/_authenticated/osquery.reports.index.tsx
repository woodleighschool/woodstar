import { createFileRoute } from "@tanstack/react-router";

import { ReportListPage } from "@/pages/osquery/reports/list";

// Pure list route: q, page, per_page, and sort are nuqs-owned.
export const Route = createFileRoute("/_authenticated/osquery/reports/")({
  component: ReportListPage,
});
