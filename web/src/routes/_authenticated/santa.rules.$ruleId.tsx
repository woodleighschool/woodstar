import { createFileRoute } from "@tanstack/react-router";

import { RuleEditPage } from "@/pages/santa/rules/edit";

export const Route = createFileRoute("/_authenticated/santa/rules/$ruleId")({
  component: RuleEditPage,
});
