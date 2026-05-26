import { createFileRoute } from "@tanstack/react-router";

import { EnrollmentsPage } from "@/pages/enrollments";

export const Route = createFileRoute("/_authenticated/enrollments/orbit")({
  component: () => <EnrollmentsPage integration="orbit" />,
});
