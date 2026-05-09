import { createFileRoute } from "@tanstack/react-router";

import { ReportEditPage } from "@/pages/reports/edit";

export const Route = createFileRoute("/_authenticated/reports/new")({
  component: () => <ReportEditPage mode="create" />,
});
