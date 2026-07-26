import { createFileRoute } from "@tanstack/react-router";

import { LabelCreatePage } from "@features/labels/create";

export const Route = createFileRoute("/_authenticated/labels/new")({
  staticData: { breadcrumb: "Create" },
  component: LabelCreatePage,
});
