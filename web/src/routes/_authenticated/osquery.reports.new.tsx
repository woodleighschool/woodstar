import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/osquery/reports/new")({
  component: lazyRouteComponent(() => import("@/pages/osquery/reports/form"), "ReportNewPage"),
});
