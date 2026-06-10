import { createFileRoute } from "@tanstack/react-router";

import { ConfigurationCreatePage } from "@/pages/santa/configurations/create";

export const Route = createFileRoute("/_authenticated/santa/configurations/new")({
  component: ConfigurationCreatePage,
});
