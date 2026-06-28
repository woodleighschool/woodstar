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
  const distributionPointId = params.distributionPointId ?? "";
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
      submitLabel="Save"
      onSubmit={async (body) => {
        const saved = await update.mutateAsync({ id: point.id, body });
        void navigate({
          to: "/munki/distribution-points/$distributionPointId",
          params: { distributionPointId: String(saved.id) },
        });
      }}
    />
  );
}
