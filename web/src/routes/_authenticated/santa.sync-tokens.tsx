import { createFileRoute } from "@tanstack/react-router";

import { SantaSyncTokensPage } from "@/pages/santa/sync-tokens";

export const Route = createFileRoute("/_authenticated/santa/sync-tokens")({
  component: SantaSyncTokensPage,
});
