import { createFileRoute } from "@tanstack/react-router";

import { MunkiSoftwareListPage } from "@/pages/munki/software/list";

// Pure list route: q, page, per_page, and sort are nuqs-owned.
export const Route = createFileRoute("/_authenticated/munki/software/")({
  component: MunkiSoftwareListPage,
});
