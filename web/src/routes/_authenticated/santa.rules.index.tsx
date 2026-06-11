import { createFileRoute } from "@tanstack/react-router";

import { RuleListPage } from "@/pages/santa/rules/list";

// Pure list route: q, page, per_page, sort, and the rule_type facet are nuqs-owned.
export const Route = createFileRoute("/_authenticated/santa/rules/")({
  component: RuleListPage,
});
