import { createFileRoute } from "@tanstack/react-router";

import { redirectForEntry } from "@features/authn/guards";

export const Route = createFileRoute("/")({
  beforeLoad: ({ context }) => redirectForEntry(context.queryClient),
});
