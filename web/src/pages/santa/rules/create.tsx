import { useNavigate, useSearch } from "@tanstack/react-router";

import { useCreateSantaRule } from "@/hooks/use-santa-rules";
import { RuleForm } from "@/pages/santa/rules/fields";
import { formFromSearch } from "@/pages/santa/rules/form-state";

export function RuleCreatePage() {
  const navigate = useNavigate();
  const search = useSearch({ strict: false });
  const create = useCreateSantaRule();

  return (
    <RuleForm
      initial={formFromSearch(search)}
      title="New Rule"
      submitLabel="Create"
      onCancel={() => void navigate({ to: "/santa/rules" })}
      onSubmit={async (body) => {
        const saved = await create.mutateAsync(body);
        void navigate({ to: "/santa/rules/$ruleId", params: { ruleId: String(saved.id) } });
      }}
    />
  );
}
