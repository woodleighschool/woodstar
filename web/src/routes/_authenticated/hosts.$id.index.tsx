import { createFileRoute } from "@tanstack/react-router";

import { HostDetailsPage } from "@features/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$id/")({
  component: HostDetailsPage,
});
