import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/labels/$labelId/edit")({
  component: lazyRouteComponent(() => import("@/pages/labels/edit"), "LabelUpdatePage"),
});
