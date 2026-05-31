import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/releases/new")({
  component: lazyRouteComponent(() => import("@/pages/munki/edit"), "MunkiReleaseNewPage"),
});
