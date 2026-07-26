import { createFileRoute } from "@tanstack/react-router";

import { HostChecksPage } from "@features/hosts/detail";

export const Route = createFileRoute("/_authenticated/hosts/$id/checks")({
  component: HostChecksPage,
});
