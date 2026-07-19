import { createFileRoute } from "@tanstack/react-router";

import { ReportEditPage } from "@/pages/osquery/reports/edit";

export const Route = createFileRoute("/_authenticated/osquery/reports/$reportId/")({
  component: ReportEditPage,
});
