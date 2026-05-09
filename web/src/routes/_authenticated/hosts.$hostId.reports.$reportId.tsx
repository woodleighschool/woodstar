import { createFileRoute } from "@tanstack/react-router";

import { HostReportResultsPage } from "@/pages/hosts/report-results";

export const Route = createFileRoute("/_authenticated/hosts/$hostId/reports/$reportId")({
  component: HostReportResultsPage,
});
