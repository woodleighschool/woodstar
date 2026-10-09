import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";

import { DistributionPointForm, formFromDistributionPoint } from "./fields";
import { useMunkiDistributionPoint, useUpdateMunkiDistributionPoint } from "./queries";

export function DistributionPointEditPage() {
  const navigate = useNavigate();
  const { id } = useParams({ from: "/_authenticated/munki/distribution-points/$id" });
  const detail = useMunkiDistributionPoint(id);
  const update = useUpdateMunkiDistributionPoint();

  if (!detail.data) {
    return (
      <QueryGate
        title="Failed to load distribution point"
        error={detail.error}
        onRetry={() => void detail.refetch()}
      />
    );
  }

  const point = detail.data;
  return (
    <DistributionPointForm
      key={point.id}
      initial={formFromDistributionPoint(point)}
      title="Edit Distribution Point"
      submitLabel="Save"
      onCancel={() =>
        void navigate({
          to: "/munki/distribution-points/$id",
          params: { id: point.id },
        })
      }
      onSubmit={async (body) => (await update.mutateAsync({ id: point.id, body })).id}
      onSuccess={(savedID) => {
        if (savedID === undefined) return;
        void navigate({
          to: "/munki/distribution-points/$id",
          params: { id: savedID },
        });
      }}
    />
  );
}
