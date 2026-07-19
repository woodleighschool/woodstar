import { createFileRoute } from "@tanstack/react-router";

import { redirectCompletedSetup } from "@/lib/auth-router";
import { SetupPage } from "@/pages/setup";

export const Route = createFileRoute("/setup")({
  beforeLoad: ({ context }) => redirectCompletedSetup(context.queryClient),
  component: SetupPage,
});
