import { createFileRoute } from "@tanstack/react-router";

import { ReportDetailPage } from "@features/osquery/reports/detail";

export const Route = createFileRoute("/_authenticated/osquery/reports/$id/")({
  component: ReportDetailPage,
});
