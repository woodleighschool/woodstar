import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/enrollments/")({
  beforeLoad: () => {
    throw redirect({ to: "/enrollments/orbit" });
  },
});
