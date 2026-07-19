import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/santa/rules")({
  staticData: { breadcrumb: "Rules" },
});
