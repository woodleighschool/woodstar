import { createFileRoute } from "@tanstack/react-router";

import { SantaEnrollmentsPage } from "@/pages/enrollments/santa";

export const Route = createFileRoute("/_authenticated/enrollments/santa")({
  staticData: { breadcrumb: "Santa" },
  component: SantaEnrollmentsPage,
});
