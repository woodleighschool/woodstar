import { createFileRoute } from "@tanstack/react-router";

import { requireAdmin } from "@/lib/auth-router";
import { UsersPage } from "@/pages/users";

export const Route = createFileRoute("/_authed/users")({
  beforeLoad: requireAdmin,
  component: UsersPage,
});
