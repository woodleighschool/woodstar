import { createFileRoute } from "@tanstack/react-router";

import { ReportLivePage } from "@features/osquery/live/page";

export const Route = createFileRoute("/_authenticated/osquery/reports/new/live")({
  staticData: { breadcrumb: "Live" },
  component: ReportLivePage,
});
