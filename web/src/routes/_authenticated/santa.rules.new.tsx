import { createFileRoute } from "@tanstack/react-router";

import { SantaRuleEditPage } from "@/pages/santa/rules";

export const Route = createFileRoute("/_authenticated/santa/rules/new")({
  component: () => <SantaRuleEditPage mode="create" />,
});
