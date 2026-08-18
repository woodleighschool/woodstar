import { createFileRoute } from "@tanstack/react-router";

import { ActivityPage } from "@features/activity/page";

export const Route = createFileRoute("/_authenticated/activity")({
  staticData: { breadcrumb: "Activity" },
  component: ActivityPage,
});
