import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/santa/configurations/$id/rules")({
  staticData: { breadcrumb: "Rules" },
});
