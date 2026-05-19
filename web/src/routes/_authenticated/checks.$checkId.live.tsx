import { createFileRoute } from "@tanstack/react-router";

import { CheckLivePage } from "@/pages/checks/live";

export const Route = createFileRoute("/_authenticated/checks/$checkId/live")({
  component: CheckLivePage,
});
