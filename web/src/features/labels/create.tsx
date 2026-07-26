import { useNavigate } from "@tanstack/react-router";

import { emptyLabel, LabelForm } from "@features/labels/fields";
import { useCreateLabel } from "@features/labels/queries";

export function LabelCreatePage() {
  const navigate = useNavigate();
  const create = useCreateLabel();

  return (
    <LabelForm
      initial={emptyLabel}
      title="Create Label"
      submitLabel="Create"
      onCancel={() => void navigate({ to: "/labels" })}
      onSubmit={async (body) => (await create.mutateAsync(body)).id}
      onSuccess={(id) => {
        if (id !== undefined) {
          void navigate({ to: "/labels/$id", params: { id: String(id) } });
        }
      }}
    />
  );
}
