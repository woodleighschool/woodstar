import { getRouteApi } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";

import { RuleForm } from "./fields";
import { formFromRule } from "./form-state";
import { useSantaRule, useUpdateSantaRule } from "./queries";

const routeApi = getRouteApi("/_authenticated/santa/rules/$id/edit");

export function RuleEditPage() {
  const navigate = routeApi.useNavigate();
  const search = routeApi.useSearch();
  const { id } = routeApi.useParams();
  const detail = useSantaRule(id);
  const update = useUpdateSantaRule();

  if (!detail.data) {
    return (
      <QueryGate
        title="Failed to Load Rule"
        error={detail.error}
        onRetry={() => void detail.refetch()}
      />
    );
  }

  const rule = detail.data;
  return (
    <RuleForm
      key={rule.id}
      initial={formFromRule(rule)}
      title="Edit Rule"
      submitLabel="Save"
      activeTab={search.tab ?? "options"}
      onActiveTabChange={(value) =>
        void navigate({
          search: (previous) => ({
            ...previous,
            tab: value === "targets" ? "targets" : undefined,
          }),
        })
      }
      onCancel={() =>
        void navigate({
          to: "/santa/rules/$id",
          params: { id: rule.id },
        })
      }
      onSubmit={async (body) => (await update.mutateAsync({ id: rule.id, body })).id}
      onSuccess={(savedID) => {
        if (savedID !== undefined) {
          void navigate({
            to: "/santa/rules/$id",
            params: { id: savedID },
          });
        }
      }}
    />
  );
}
