import { createFileRoute, Outlet } from "@tanstack/react-router";

import { requireAdmin } from "@/lib/auth-router";

export const Route = createFileRoute("/_authenticated/directory")({
  beforeLoad: requireAdmin,
  component: () => <Outlet />,
});
