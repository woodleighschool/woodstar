import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/packages/$packageId/edit")({
  component: lazyRouteComponent(() => import("@/pages/munki/packages/edit"), "MunkiPackageEditPage"),
});
