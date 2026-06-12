import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/santa/")({
  beforeLoad: () => {
    throw redirect({ to: "/santa/configurations" });
  },
});
