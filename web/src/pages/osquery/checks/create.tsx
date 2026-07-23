import { useNavigate } from "@tanstack/react-router";

import { useCreateCheck } from "@/hooks/use-checks";
import { CheckForm, emptyCheck } from "@/pages/osquery/checks/fields";

export function CheckCreatePage() {
  const navigate = useNavigate();
  const create = useCreateCheck();

  return (
    <CheckForm
      initial={emptyCheck}
      title="Create Check"
      submitLabel="Create"
      onCancel={() => void navigate({ to: "/osquery/checks" })}
      onSubmit={async (value) => (await create.mutateAsync(value)).id}
      onSuccess={(id) => {
        if (id !== undefined) {
          void navigate({ to: "/osquery/checks/$id", params: { id: String(id) } });
        }
      }}
    />
  );
}
