import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/software-titles/new")({
  component: lazyRouteComponent(() => import("@/pages/munki/software-title/new"), "MunkiSoftwareTitleNewPage"),
});
