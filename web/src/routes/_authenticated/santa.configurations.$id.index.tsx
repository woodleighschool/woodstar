import { createFileRoute } from "@tanstack/react-router";

import { ConfigurationDetailPage } from "@features/santa/configurations/detail";

export const Route = createFileRoute("/_authenticated/santa/configurations/$id/")({
  component: ConfigurationDetailPage,
});
