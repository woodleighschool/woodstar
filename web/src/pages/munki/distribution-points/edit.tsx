import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@/components/query-gate";
import {
  useMunkiDistributionPoint,
  useUpdateMunkiDistributionPoint,
} from "@/hooks/use-munki-distribution-points";
import { parseRouteID } from "@/lib/route-params";
import {
  DistributionPointForm,
  formFromDistributionPoint,
} from "@/pages/munki/distribution-points/fields";

export function DistributionPointEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const distributionPointId = params.id ?? "";
  const id = parseRouteID(distributionPointId);
  const detail = useMunkiDistributionPoint(id);
  const update = useUpdateMunkiDistributionPoint();

  if (id === null) {
    return (
      <QueryGate
        title="Failed to load distribution point"
        error={{ message: "Distribution point route is invalid." }}
      />
    );
  }

  if (detail.error || !detail.data) {
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
      onSubmit={async (body) => (await update.mutateAsync({ id: point.id, body })).id}
      onSuccess={(savedID) => {
        if (savedID === undefined) return;
        void navigate({
          to: "/munki/distribution-points/$id",
          params: { id: String(savedID) },
        });
      }}
    />
  );
}
