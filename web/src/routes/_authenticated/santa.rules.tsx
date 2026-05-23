import { createFileRoute } from "@tanstack/react-router";

import { SantaRulesPage } from "@/pages/santa/rules";

export const Route = createFileRoute("/_authenticated/santa/rules")({
  component: SantaRulesPage,
});
