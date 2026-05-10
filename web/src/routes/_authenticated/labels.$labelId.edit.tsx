import { createFileRoute } from "@tanstack/react-router";

import { LabelEditPage } from "@/pages/labels/edit";

export const Route = createFileRoute("/_authenticated/labels/$labelId/edit")({
  component: () => <LabelEditPage mode="edit" />,
});
