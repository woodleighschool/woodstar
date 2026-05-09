import { createFileRoute } from "@tanstack/react-router";

import { ReportsPage } from "@/pages/reports/list";

export const Route = createFileRoute("/_authenticated/reports/")({
  component: ReportsPage,
});
