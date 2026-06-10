import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/osquery/checks/new")({
  component: lazyRouteComponent(() => import("@/pages/osquery/checks/create"), "CheckCreatePage"),
});
