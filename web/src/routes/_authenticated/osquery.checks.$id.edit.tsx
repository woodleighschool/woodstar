import { createFileRoute } from "@tanstack/react-router";

import { CheckEditPage } from "@features/osquery/checks/edit";

export const Route = createFileRoute("/_authenticated/osquery/checks/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  component: CheckEditPage,
});
