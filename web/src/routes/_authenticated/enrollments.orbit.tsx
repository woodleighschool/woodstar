import { createFileRoute } from "@tanstack/react-router";

import { OrbitEnrollmentsPage } from "@features/enrollments/orbit";

export const Route = createFileRoute("/_authenticated/enrollments/orbit")({
  staticData: { breadcrumb: "Orbit" },
  component: OrbitEnrollmentsPage,
});
