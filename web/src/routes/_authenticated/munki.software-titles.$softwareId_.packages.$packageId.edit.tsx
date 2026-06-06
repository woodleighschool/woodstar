import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/software-titles/$softwareId_/packages/$packageId/edit")({
  component: lazyRouteComponent(() => import("@/pages/munki/software-title/package-edit"), "MunkiPackageEditPage"),
});
