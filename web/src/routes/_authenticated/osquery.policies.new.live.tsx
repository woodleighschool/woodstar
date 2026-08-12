import { createFileRoute } from "@tanstack/react-router";

import { PolicyLivePage } from "@features/osquery/live/page";

export const Route = createFileRoute("/_authenticated/osquery/policies/new/live")({
  staticData: { breadcrumb: "Live" },
  component: PolicyLivePage,
});
