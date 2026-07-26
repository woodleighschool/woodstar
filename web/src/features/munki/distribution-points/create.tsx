import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { DistributionPointForm, emptyDistributionPointForm } from "./fields";
import { KeyRevealDialog } from "./key-reveal-dialog";
import { useCreateMunkiDistributionPoint } from "./queries";

export function DistributionPointCreatePage() {
  const navigate = useNavigate();
  const create = useCreateMunkiDistributionPoint();
  const [created, setCreated] = useState<{ id: number; key: string } | null>(null);

  return (
    <>
      <DistributionPointForm
        initial={emptyDistributionPointForm}
        title="Create Distribution Point"
        submitLabel="Create"
        onCancel={() => void navigate({ to: "/munki/distribution-points" })}
        onSubmit={async (body) => {
          const saved = await create.mutateAsync(body);
          setCreated({ id: saved.id, key: saved.key });
          return saved.id;
        }}
      />

      {created ? (
        <KeyRevealDialog
          title="Distribution Point Key"
          description="Copy this key into the worker configuration. It is shown only once."
          value={created.key}
          open
          onOpenChange={(open) => {
            if (open) return;
            const id = created.id;
            setCreated(null);
            void navigate({
              to: "/munki/distribution-points/$id",
              params: { id: String(id) },
            });
          }}
        />
      ) : null}
    </>
  );
}
