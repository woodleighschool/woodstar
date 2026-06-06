import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/osquery/reports/$reportId/")({
  component: lazyRouteComponent(() => import("@/pages/osquery/reports/form"), "ReportResourcePage"),
});
