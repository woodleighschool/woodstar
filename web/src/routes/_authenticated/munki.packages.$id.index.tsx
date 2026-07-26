import { createFileRoute } from "@tanstack/react-router";

import { MunkiPackageDetailPage } from "@features/munki/packages/detail";

export const Route = createFileRoute("/_authenticated/munki/packages/$id/")({
  component: MunkiPackageDetailPage,
});
