import { createFileRoute } from "@tanstack/react-router";

import { ReportLivePage } from "@/pages/osquery/reports/live";

export const Route = createFileRoute("/_authenticated/osquery/reports/$reportId/live")({
  component: ReportLivePage,
});
