import { createFileRoute } from "@tanstack/react-router";

import { tableSearchSchema } from "@/lib/pagination";
import { ReportListPage } from "@/pages/osquery/reports/list";

export const Route = createFileRoute("/_authenticated/osquery/reports/")({
  validateSearch: (search) => tableSearchSchema.parse(search),
  component: ReportListPage,
});
