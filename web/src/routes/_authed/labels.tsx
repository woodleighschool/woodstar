import { createFileRoute } from "@tanstack/react-router";

import { LabelsPage } from "@/pages/labels";

export const Route = createFileRoute("/_authed/labels")({
  component: LabelsPage,
});
