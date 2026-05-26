import { createFileRoute, Navigate } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/enrollments/")({
  component: () => <Navigate to="/enrollments/orbit" replace />,
});
