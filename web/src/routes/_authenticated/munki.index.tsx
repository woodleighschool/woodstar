import { createFileRoute, Navigate } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/")({
  component: () => <Navigate to="/munki/software" replace />,
});
