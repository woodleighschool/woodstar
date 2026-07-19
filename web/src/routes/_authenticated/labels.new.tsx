import { createFileRoute } from "@tanstack/react-router";

import { LabelCreatePage } from "@/pages/labels/create";

export const Route = createFileRoute("/_authenticated/labels/new")({
  staticData: { breadcrumb: "New" },
  component: LabelCreatePage,
});
