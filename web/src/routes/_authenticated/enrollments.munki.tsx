import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/enrollments/munki")({
  component: lazyRouteComponent(() => import("@/pages/enrollments/munki"), "MunkiEnrollmentsPage"),
});
