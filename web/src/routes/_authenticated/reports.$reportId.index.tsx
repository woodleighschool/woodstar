import { createFileRoute } from "@tanstack/react-router";

import { ReportDetailPage } from "@/pages/reports/detail";

export const Route = createFileRoute("/_authenticated/reports/$reportId/")({
  component: ReportDetailPage,
});
