import { createFileRoute } from "@tanstack/react-router";

import { CheckLivePage } from "@features/osquery/checks/live";

export const Route = createFileRoute("/_authenticated/osquery/checks/$id/live")({
  staticData: { breadcrumb: "Live" },
  component: CheckLivePage,
});
