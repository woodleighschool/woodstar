import { createFileRoute, Outlet } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/osquery/checks/$checkId")({
  component: () => <Outlet />,
});
