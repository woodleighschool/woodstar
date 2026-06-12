import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/directory/")({
  beforeLoad: () => {
    throw redirect({ to: "/directory/users" });
  },
});
