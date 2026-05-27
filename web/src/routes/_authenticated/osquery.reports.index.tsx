import { createFileRoute } from "@tanstack/react-router";

import { ReportsPage } from "@/pages/osquery/reports/list";

export const Route = createFileRoute("/_authenticated/osquery/reports/")({
  component: ReportsPage,
});
