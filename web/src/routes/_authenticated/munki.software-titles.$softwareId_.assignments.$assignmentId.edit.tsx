import { createFileRoute, lazyRouteComponent } from "@tanstack/react-router";

export const Route = createFileRoute(
  "/_authenticated/munki/software-titles/$softwareId_/assignments/$assignmentId/edit",
)({
  component: lazyRouteComponent(() => import("@/pages/munki/assignment-edit"), "MunkiAssignmentEditPage"),
});
