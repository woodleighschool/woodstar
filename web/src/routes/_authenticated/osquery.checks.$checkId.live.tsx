import { createFileRoute } from "@tanstack/react-router";

import { CheckLivePage } from "@/pages/osquery/checks/live";

export const Route = createFileRoute("/_authenticated/osquery/checks/$checkId/live")({
  component: CheckLivePage,
});
