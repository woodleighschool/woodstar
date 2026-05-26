import { createFileRoute } from "@tanstack/react-router";

import { EnrollmentsPage } from "@/pages/enrollments";

export const Route = createFileRoute("/_authenticated/enrollments/santa")({
  component: () => <EnrollmentsPage integration="santa" />,
});
