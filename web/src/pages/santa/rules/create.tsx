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
      title="Create Rule"
      submitLabel="Create"
      onCancel={() => void navigate({ to: "/santa/rules" })}
      onSubmit={async (body) => (await create.mutateAsync(body)).id}
      onSuccess={(id) => {
        if (id !== undefined) {
          void navigate({ to: "/santa/rules/$id", params: { id: String(id) } });
        }
      }}
    />
  );
}
