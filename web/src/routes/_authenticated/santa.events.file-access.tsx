import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/santa/events/file-access")({
  component: () => <Outlet />,
});
