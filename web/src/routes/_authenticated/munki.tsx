import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki")({
  staticData: { breadcrumb: "Munki" },
});
