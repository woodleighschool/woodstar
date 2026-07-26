import { createFileRoute } from "@tanstack/react-router";

import { SantaEnrollmentsPage } from "@features/enrollments/santa";

export const Route = createFileRoute("/_authenticated/enrollments/santa")({
  staticData: { breadcrumb: "Santa" },
  component: SantaEnrollmentsPage,
});
