import { createFileRoute, Navigate } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/santa/")({
  component: () => <Navigate to="/santa/configurations" replace />,
});
