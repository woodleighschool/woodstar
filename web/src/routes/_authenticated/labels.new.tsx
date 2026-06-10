import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/labels/new")({
  component: lazyRouteComponent(() => import("@/pages/labels/create"), "LabelCreatePage"),
});
