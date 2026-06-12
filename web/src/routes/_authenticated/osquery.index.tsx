import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/osquery/")({
  beforeLoad: () => {
    throw redirect({ to: "/osquery/reports" });
  },
});
