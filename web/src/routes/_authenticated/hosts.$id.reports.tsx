import { createFileRoute } from "@tanstack/react-router";

import { HostReportsPage } from "@features/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$id/reports")({
  component: HostReportsPage,
});
