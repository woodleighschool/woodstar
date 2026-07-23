import { createFileRoute } from "@tanstack/react-router";

import { MunkiSoftwareCreatePage } from "@/pages/munki/software/create";

export const Route = createFileRoute("/_authenticated/munki/software/new")({
  staticData: { breadcrumb: "Create" },
  component: MunkiSoftwareCreatePage,
});
