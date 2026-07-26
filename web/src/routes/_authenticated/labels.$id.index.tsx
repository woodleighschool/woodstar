import { createFileRoute } from "@tanstack/react-router";

import { LabelDetailPage } from "@features/labels/detail";

export const Route = createFileRoute("/_authenticated/labels/$id/")({
  component: LabelDetailPage,
});
