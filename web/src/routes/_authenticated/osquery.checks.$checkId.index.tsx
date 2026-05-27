import { createFileRoute } from "@tanstack/react-router";

import { CheckDetailPage } from "@/pages/osquery/checks/detail";

export const Route = createFileRoute("/_authenticated/osquery/checks/$checkId/")({
  component: CheckDetailPage,
});
