import { useNavigate, useParams } from "@tanstack/react-router";

import { PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import {
  useMunkiDistributionPoint,
  useUpdateMunkiDistributionPoint,
} from "@/hooks/use-munki-distribution-points";
import {
  DistributionPointForm,
  formFromDistributionPoint,
} from "@/pages/munki/distribution-points/fields";

export function DistributionPointEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const distributionPointId = params.distributionPointId ?? "";
  const id = Number(distributionPointId);
  const detail = useMunkiDistributionPoint(id);
  const update = useUpdateMunkiDistributionPoint();

  if (detail.error) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load distribution point"
          error={detail.error}
          onRetry={() => void detail.refetch()}
        />
      </PageShell>
    );
  }
  if (!detail.data) return null;

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
