import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/osquery/reports")({
  staticData: { breadcrumb: "Reports" },
});
