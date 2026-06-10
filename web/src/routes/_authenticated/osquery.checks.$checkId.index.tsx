import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/osquery/checks/$checkId/")({
  component: lazyRouteComponent(() => import("@/pages/osquery/checks/edit"), "CheckEditPage"),
});
