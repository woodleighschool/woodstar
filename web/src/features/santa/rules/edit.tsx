import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { parseRouteID } from "@lib/route-params";

import { RuleForm } from "./fields";
import { formFromRule } from "./form-state";
import { useSantaRule, useUpdateSantaRule } from "./queries";

export function RuleEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const ruleId = params.id ?? "";
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
      onCancel={() =>
        void navigate({
          to: "/santa/rules/$id",
          params: { id: String(rule.id) },
        })
      }
      onSubmit={async (body) => (await update.mutateAsync({ id: rule.id, body })).id}
      onSuccess={(savedID) => {
        if (savedID !== undefined) {
          void navigate({
            to: "/santa/rules/$id",
            params: { id: String(savedID) },
          });
        }
      }}
    />
  );
}
