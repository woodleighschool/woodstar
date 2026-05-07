import { createFileRoute } from "@tanstack/react-router";

import { HostDetailPage } from "@/pages/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$hostId")({
  component: HostDetailPage,
});
