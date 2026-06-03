import { createFileRoute, Navigate } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/directory/")({
  component: () => <Navigate to="/directory/users" replace />,
});
