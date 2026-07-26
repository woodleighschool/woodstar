import { createFileRoute } from "@tanstack/react-router";

import { redirectForEntry } from "@features/auth/guards";

export const Route = createFileRoute("/")({
  beforeLoad: ({ context }) => redirectForEntry(context.queryClient),
});
