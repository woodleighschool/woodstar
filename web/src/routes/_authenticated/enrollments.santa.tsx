import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/enrollments/santa")({
  component: lazyRouteComponent(() => import("@/pages/enrollments"), "SantaEnrollmentsPage"),
});
