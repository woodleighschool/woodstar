import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/assignments/new")({
  component: lazyRouteComponent(() => import("@/pages/munki/edit"), "MunkiAssignmentNewPage"),
});
