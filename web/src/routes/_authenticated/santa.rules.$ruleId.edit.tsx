import { createFileRoute } from "@tanstack/react-router";

import { SantaRuleEditPage } from "@/pages/santa/rules/edit";

export const Route = createFileRoute("/_authenticated/santa/rules/$ruleId/edit")({
  component: () => <SantaRuleEditPage mode="edit" />,
});
