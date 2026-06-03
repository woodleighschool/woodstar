import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/directory/users")({
  component: () => <Outlet />,
});
