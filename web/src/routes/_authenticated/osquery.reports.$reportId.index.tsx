import { createFileRoute } from "@tanstack/react-router";

import { ReportDetailPage } from "@/pages/osquery/reports/detail";

export const Route = createFileRoute("/_authenticated/osquery/reports/$reportId/")({
  component: ReportDetailPage,
});
