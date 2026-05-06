import { createFileRoute } from "@tanstack/react-router";

import { HostDetailPage } from "@/pages/hosts/detail";

export const Route = createFileRoute("/_authed/hosts/$hostId")({
  component: HostDetailPage,
});
