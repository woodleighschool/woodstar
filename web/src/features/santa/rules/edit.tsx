import { getRouteApi } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { parseRouteID } from "@lib/route-params";

import { RuleForm } from "./fields";
import { formFromRule } from "./form-state";
import { useSantaRule, useUpdateSantaRule } from "./queries";

const routeApi = getRouteApi("/_authenticated/santa/configurations/$id/rules/$ruleId/edit");

export function RuleEditPage() {
  const navigate = routeApi.useNavigate();
  const search = routeApi.useSearch();
  const params = routeApi.useParams();
  const configurationID = parseRouteID(params.id);
  const ruleID = parseRouteID(params.ruleId);
  const detail = useSantaRule(ruleID);
  const update = useUpdateSantaRule();

  if (configurationID === null || ruleID === null) {
    return <QueryGate title="Failed to load rule" error={{ message: "Rule route is invalid." }} />;
  }

  if (detail.error || !detail.data) {
    return (
      <QueryGate
        title="Failed to load rule"
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
          to: "/santa/configurations/$id/rules/$ruleId",
          params: { id: String(configurationID), ruleId: String(rule.id) },
        })
      }
      onSubmit={async (body) =>
        (
          await update.mutateAsync({
            id: rule.id,
            body: { ...body, configuration_id: configurationID },
          })
        ).id
      }
      onSuccess={(savedID) => {
        if (savedID !== undefined) {
          void navigate({
            to: "/santa/configurations/$id/rules/$ruleId",
            params: { id: String(configurationID), ruleId: String(savedID) },
          });
        }
      }}
    />
  );
}
