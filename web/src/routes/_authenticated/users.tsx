import { createFileRoute } from "@tanstack/react-router";

import { requireAdmin } from "@/lib/auth-router";
import { UsersPage } from "@/pages/users";

export const Route = createFileRoute("/_authenticated/users")({
  beforeLoad: requireAdmin,
  component: UsersPage,
});
