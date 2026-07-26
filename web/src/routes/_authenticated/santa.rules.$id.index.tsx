import { createFileRoute } from "@tanstack/react-router";

import { RuleDetailPage } from "@features/santa/rules/detail";

export const Route = createFileRoute("/_authenticated/santa/rules/$id/")({
  component: RuleDetailPage,
});
