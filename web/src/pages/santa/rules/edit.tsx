import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@/components/query-gate";
import { useSantaRule, useUpdateSantaRule } from "@/hooks/use-santa-rules";
import { parseRouteID } from "@/lib/route-params";
import { RuleForm } from "@/pages/santa/rules/fields";
import { formFromRule } from "@/pages/santa/rules/form-state";

export function RuleEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const ruleId = params.ruleId ?? "";
  const id = parseRouteID(ruleId);
  const detail = useSantaRule(id);
  const update = useUpdateSantaRule();

  if (id === null) {
    return <QueryGate title="Failed to load rule" error={{ message: "Rule route is invalid." }} />;
  }

  if (detail.error || !detail.data) {
    return (
      <QueryGate
        title="Failed to load rule"
        error={detail.error}
        onRetry={() => void detail.refetch()}
      />
    );
  }

  const rule = detail.data;
  return (
    <RuleForm
      key={rule.id}
      initial={formFromRule(rule)}
      title="Edit Rule"
      submitLabel="Save"
      onSubmit={async (body) => (await update.mutateAsync({ id: rule.id, body })).id}
      onSuccess={(savedID) => {
        if (savedID !== undefined) {
          void navigate({ to: "/santa/rules/$ruleId", params: { ruleId: String(savedID) } });
        }
      }}
    />
  );
}
