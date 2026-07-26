import { createFileRoute } from "@tanstack/react-router";

import { ReportEditPage } from "@features/osquery/reports/edit";

export const Route = createFileRoute("/_authenticated/osquery/reports/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  component: ReportEditPage,
});
