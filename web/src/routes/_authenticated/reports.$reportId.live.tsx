import { createFileRoute } from "@tanstack/react-router";

import { ReportLivePage } from "@/pages/reports/live";

export const Route = createFileRoute("/_authenticated/reports/$reportId/live")({
  component: ReportLivePage,
});
