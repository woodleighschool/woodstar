import { createFileRoute } from "@tanstack/react-router";

import { requireAdmin } from "@/lib/auth-router";
import { UserEditPage } from "@/pages/users/edit";

export const Route = createFileRoute("/_authenticated/users/$userId/edit")({
  beforeLoad: requireAdmin,
  component: UserEditPage,
});
