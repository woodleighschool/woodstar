import { useNavigate } from "@tanstack/react-router";

import { useCreateCheck } from "@/hooks/use-checks";
import { CheckForm, emptyCheck } from "@/pages/osquery/checks/fields";

export function CheckCreatePage() {
  const navigate = useNavigate();
  const create = useCreateCheck();

  return (
    <CheckForm
      initial={emptyCheck}
      title="New Check"
      submitLabel="Create"
      onCancel={() => void navigate({ to: "/osquery/checks" })}
      onSubmit={async (value) => {
        const saved = await create.mutateAsync(value);
        void navigate({ to: "/osquery/checks/$checkId", params: { checkId: String(saved.id) } });
      }}
    />
  );
}
