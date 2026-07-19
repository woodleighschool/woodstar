import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/hosts")({
  staticData: { breadcrumb: "Hosts" },
});
