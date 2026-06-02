import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/software-titles/$softwareId_/edit")({
  component: lazyRouteComponent(() => import("@/pages/munki/edit"), "MunkiSoftwareTitleEditPage"),
});
