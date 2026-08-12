import { getRouteApi, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import {
  useClearOsqueryHistoryState,
  useOpenOsqueryLive,
  useOsqueryHistoryState,
} from "@features/osquery/live/history";
import { parseRouteID } from "@lib/route-params";

import { PolicyForm, policyFromDetail } from "./fields";
import { usePolicy, usePolicyRemediationSource, useUpdatePolicy } from "./queries";

const routeApi = getRouteApi("/_authenticated/osquery/policies/$id/edit");

export function PolicyEditPage() {
  const navigate = routeApi.useNavigate();
  const search = routeApi.useSearch();
  const params = useParams({ strict: false });
  const policyId = params.id ?? "";
  const id = parseRouteID(policyId);
  const detail = usePolicy(id);
  const remediationSource = usePolicyRemediationSource(id);
  const update = useUpdatePolicy(id);
  const historyState = useOsqueryHistoryState();
  const openLive = useOpenOsqueryLive();
  const clearHistoryState = useClearOsqueryHistoryState();

  if (id === null) {
    return (
      <QueryGate title="Failed to load policy" error={{ message: "Policy route is invalid." }} />
    );
  }

  if (detail.error || remediationSource.error || !detail.data || !remediationSource.data) {
    return (
      <QueryGate
        title="Failed to load policy"
        error={detail.error ?? remediationSource.error}
        onRetry={() => {
          void detail.refetch();
          void remediationSource.refetch();
        }}
      />
    );
  }

  const policy = detail.data;
  const draft =
    historyState?.view === "policy-form" && historyState.id === policy.id
      ? historyState.value
      : undefined;
  return (
    <PolicyForm
      key={policy.id}
      initial={policyFromDetail(policy, remediationSource.data.script)}
      draft={draft}
      title="Edit Policy"
      submitLabel="Save"
      activeTab={search.tab ?? "options"}
      onActiveTabChange={(value) =>
        void navigate({
          search: (previous) => ({
            ...previous,
            tab: value === "targets" || value === "remediation" ? value : undefined,
          }),
        })
      }
      confirmResultReset
      onCancel={async () => {
        await clearHistoryState();
        await navigate({
          to: "/osquery/policies/$id",
          params: { id: String(policy.id) },
        });
      }}
      onRunLive={(value) =>
        openLive({
          kind: "policy",
          id: policy.id,
          sql: value.query.trim(),
          form: {
            view: "policy-form",
            id: policy.id,
            value,
          },
        })
      }
      onSubmit={async (value) => (await update.mutateAsync(value)).id}
      onSuccess={async (savedID) => {
        if (savedID !== undefined) {
          await clearHistoryState();
          await navigate({
            to: "/osquery/policies/$id",
            params: { id: String(savedID) },
          });
        }
      }}
    />
  );
}
