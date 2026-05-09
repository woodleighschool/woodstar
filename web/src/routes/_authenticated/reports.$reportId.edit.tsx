import { createFileRoute } from "@tanstack/react-router";

import { ReportEditPage } from "@/pages/reports/edit";

export const Route = createFileRoute("/_authenticated/reports/$reportId/edit")({
  component: () => <ReportEditPage mode="edit" />,
});
