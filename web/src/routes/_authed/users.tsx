import { createFileRoute } from "@tanstack/react-router";

import { UsersPage } from "@/pages/users";
import { requireAdmin } from "@/lib/auth-router";

export const Route = createFileRoute("/_authed/users")({
  beforeLoad: requireAdmin,
  component: UsersPage,
});
