import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/software-titles/$softwareId_/deployments/new")({
  component: lazyRouteComponent(() => import("@/pages/munki/edit"), "MunkiDeploymentNewPage"),
});
