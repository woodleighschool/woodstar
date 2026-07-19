import { createFileRoute } from "@tanstack/react-router";

import { AppLayout } from "@/layouts/app-layout";
import { requireUser } from "@/lib/auth-router";

export const Route = createFileRoute("/_authenticated")({
  beforeLoad: ({ context }) => requireUser(context.queryClient),
  component: AppLayout,
});
