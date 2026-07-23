import { createFileRoute } from "@tanstack/react-router";

import { ReportCreatePage } from "@/pages/osquery/reports/create";

export const Route = createFileRoute("/_authenticated/osquery/reports/new")({
  staticData: { breadcrumb: "Create" },
  component: ReportCreatePage,
});
