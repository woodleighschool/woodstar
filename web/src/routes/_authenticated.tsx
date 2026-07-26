import { createFileRoute } from "@tanstack/react-router";

import { requireUser } from "@features/auth/guards";

import { AppLayout } from "../app-layout";

export const Route = createFileRoute("/_authenticated")({
  beforeLoad: ({ context }) => requireUser(context.queryClient),
  component: AppLayout,
});
