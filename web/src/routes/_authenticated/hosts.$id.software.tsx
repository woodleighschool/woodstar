import { createFileRoute } from "@tanstack/react-router";

import { HostSoftwarePage } from "@features/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$id/software")({
  component: HostSoftwarePage,
});
