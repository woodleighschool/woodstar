import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@/components/query-gate";
import { useSantaRule, useUpdateSantaRule } from "@/hooks/use-santa-rules";
import { RuleForm } from "@/pages/santa/rules/fields";
import { formFromRule } from "@/pages/santa/rules/form-state";

export function RuleEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const ruleId = params.ruleId ?? "";
  const id = Number(ruleId);
  const detail = useSantaRule(id);
  const update = useUpdateSantaRule();

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
      submitLabel="Save"
      onSubmit={async (body) => {
        const saved = await update.mutateAsync({ id: rule.id, body });
        void navigate({ to: "/santa/rules/$ruleId", params: { ruleId: String(saved.id) } });
      }}
    />
  );
}
