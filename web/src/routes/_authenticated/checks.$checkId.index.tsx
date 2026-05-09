import { createFileRoute } from "@tanstack/react-router";

import { CheckDetailPage } from "@/pages/checks/detail";

export const Route = createFileRoute("/_authenticated/checks/$checkId/")({
  component: CheckDetailPage,
});
