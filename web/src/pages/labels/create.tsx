import { useNavigate } from "@tanstack/react-router";

import { useCreateLabel } from "@/hooks/use-labels";
import { emptyLabel, LabelForm } from "@/pages/labels/fields";

export function LabelCreatePage() {
  const navigate = useNavigate();
  const create = useCreateLabel();

  return (
    <LabelForm
      initial={emptyLabel}
      title="New Label"
      submitLabel="Create"
      onCancel={() => void navigate({ to: "/labels" })}
      onSubmit={async (body) => (await create.mutateAsync(body)).id}
      onSuccess={() => void navigate({ to: "/labels" })}
    />
  );
}
