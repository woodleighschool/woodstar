import { createFileRoute } from "@tanstack/react-router";

import { LabelEditPage } from "@/pages/labels/edit";

export const Route = createFileRoute("/_authenticated/labels/new")({
  component: () => <LabelEditPage mode="create" />,
});
