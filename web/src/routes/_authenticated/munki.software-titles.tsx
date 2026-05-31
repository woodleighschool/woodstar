import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/munki/software-titles")({
  component: () => <Outlet />,
});
