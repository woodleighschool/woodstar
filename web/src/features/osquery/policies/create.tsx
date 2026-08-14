import { getRouteApi } from "@tanstack/react-router";

import {
  useClearOsqueryHistoryState,
  useOpenOsqueryLive,
  useOsqueryHistoryState,
} from "@features/osquery/live/history";

import { PolicyForm, emptyPolicy } from "./fields";
import { useCreatePolicy } from "./queries";

const routeApi = getRouteApi("/_authenticated/osquery/policies/new/");

export function PolicyCreatePage() {
  const navigate = routeApi.useNavigate();
  const search = routeApi.useSearch();
  const create = useCreatePolicy();
  const historyState = useOsqueryHistoryState();
  const openLive = useOpenOsqueryLive();
  const clearHistoryState = useClearOsqueryHistoryState();
  const draft =
    historyState?.view === "policy-form" && historyState.id === undefined
      ? historyState.value
      : undefined;

  return (
    <PolicyForm
      initial={emptyPolicy}
      draft={draft}
      title="Create Policy"
      submitLabel="Create"
      activeTab={search.tab ?? "options"}
      onActiveTabChange={(value) =>
        void navigate({
          search: (previous) => ({
            ...previous,
            tab: value === "targets" || value === "remediation" ? value : undefined,
          }),
        })
      }
      onCancel={async () => {
        await clearHistoryState();
        await navigate({ to: "/osquery/policies" });
      }}
      onRunLive={(value) =>
        openLive({
          kind: "policy",
          sql: value.query.trim(),
          form: {
            view: "policy-form",
            value,
          },
        })
      }
      onSubmit={async (value) => (await create.mutateAsync(value)).id}
      onSuccess={async (id) => {
        if (id !== undefined) {
          await clearHistoryState();
          await navigate({
            to: "/osquery/policies/$id",
            params: { id: String(id) },
          });
        }
      }}
    />
  );
}
