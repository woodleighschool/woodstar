import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/")({
  beforeLoad: () => {
    throw redirect({ to: "/munki/software" });
  },
});
