import { getRouteApi, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { parseRouteID } from "@lib/route-params";

import { RuleForm } from "./fields";
import { formFromRule } from "./form-state";
import { useSantaRule, useUpdateSantaRule } from "./queries";

const routeApi = getRouteApi("/_authenticated/santa/rules/$id/edit");

export function RuleEditPage() {
  const navigate = routeApi.useNavigate();
  const search = routeApi.useSearch();
  const params = useParams({ strict: false });
  const ruleId = params.id ?? "";
  const id = parseRouteID(ruleId);
  const detail = useSantaRule(id);
  const update = useUpdateSantaRule();

  if (id === null) {
    return <QueryGate title="Failed to Load Rule" error={{ message: "Rule route is invalid." }} />;
  }

  if (detail.error || !detail.data) {
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
          params: { id: String(rule.id) },
        })
      }
      onSubmit={async (body) => (await update.mutateAsync({ id: rule.id, body })).id}
      onSuccess={(savedID) => {
        if (savedID !== undefined) {
          void navigate({
            to: "/santa/rules/$id",
            params: { id: String(savedID) },
          });
        }
      }}
    />
  );
}
