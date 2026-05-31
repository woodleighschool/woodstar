import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/artifacts/new")({
  component: lazyRouteComponent(() => import("@/pages/munki/edit"), "MunkiArtifactNewPage"),
});
