import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/software-titles/$softwareId_/packages/new")({
  component: lazyRouteComponent(() => import("@/pages/munki/package-edit"), "MunkiPackageNewPage"),
});
