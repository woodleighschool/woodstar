import { createFileRoute } from "@tanstack/react-router";

import { SantaRuleResourcePage } from "@/pages/santa/rules/form";

export const Route = createFileRoute("/_authenticated/santa/rules/$ruleId")({
  component: () => <SantaRuleResourcePage mode="edit" />,
});
