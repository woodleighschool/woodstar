import { createFileRoute } from "@tanstack/react-router";

import { ConfigurationCreatePage } from "@features/santa/configurations/create";

export const Route = createFileRoute("/_authenticated/santa/configurations/new")({
  staticData: { breadcrumb: "Create" },
  component: ConfigurationCreatePage,
});
