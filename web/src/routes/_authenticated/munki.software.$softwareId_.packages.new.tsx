import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/software/$softwareId_/packages/new")({
  component: lazyRouteComponent(() => import("@/pages/munki/software/package-edit"), "MunkiPackageNewPage"),
});
