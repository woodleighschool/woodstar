import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/checks/$checkId")({
  component: () => <Outlet />,
});
