import { createFileRoute } from "@tanstack/react-router";

import { HostSantaPage } from "@features/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$id/santa")({
  component: HostSantaPage,
});
