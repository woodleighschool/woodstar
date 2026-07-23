import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { useCreateMunkiDistributionPoint } from "@/hooks/use-munki-distribution-points";
import {
  DistributionPointForm,
  emptyDistributionPointForm,
} from "@/pages/munki/distribution-points/fields";
import { KeyRevealDialog } from "@/pages/munki/distribution-points/key-reveal-dialog";

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
              to: "/munki/distribution-points/$distributionPointId",
              params: { distributionPointId: String(id) },
            });
          }}
        />
      ) : null}
    </>
  );
}
