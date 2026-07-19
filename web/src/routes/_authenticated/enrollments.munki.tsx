import { createFileRoute } from "@tanstack/react-router";

import { MunkiEnrollmentsPage } from "@/pages/enrollments/munki";

export const Route = createFileRoute("/_authenticated/enrollments/munki")({
  staticData: { breadcrumb: "Munki" },
  component: MunkiEnrollmentsPage,
});
