import { createFileRoute, Navigate } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/osquery/")({
  component: () => <Navigate to="/osquery/reports" replace />,
});
