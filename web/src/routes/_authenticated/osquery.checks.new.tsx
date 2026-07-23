import { createFileRoute } from "@tanstack/react-router";

import { CheckCreatePage } from "@/pages/osquery/checks/create";

export const Route = createFileRoute("/_authenticated/osquery/checks/new")({
  staticData: { breadcrumb: "Create" },
  component: CheckCreatePage,
});
