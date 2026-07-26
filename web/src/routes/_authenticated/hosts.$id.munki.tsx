import { createFileRoute } from "@tanstack/react-router";

import { HostMunkiPage } from "@features/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$id/munki")({
  component: HostMunkiPage,
});
