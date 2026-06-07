import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/software/new")({
  component: lazyRouteComponent(() => import("@/pages/munki/software/new"), "MunkiSoftwareNewPage"),
});
