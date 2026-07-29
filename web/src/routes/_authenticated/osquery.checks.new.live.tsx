import { createFileRoute } from "@tanstack/react-router";

import { CheckLivePage } from "@features/osquery/live/page";

export const Route = createFileRoute("/_authenticated/osquery/checks/new/live")({
  staticData: { breadcrumb: "Live" },
  component: CheckLivePage,
});
