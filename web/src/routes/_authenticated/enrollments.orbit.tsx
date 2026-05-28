import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/enrollments/orbit")({
  component: lazyRouteComponent(() => import("@/pages/enrollments/orbit"), "OrbitEnrollmentsPage"),
});
