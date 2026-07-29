import { getRouteApi } from "@tanstack/react-router";

import {
  useClearOsqueryHistoryState,
  useOpenOsqueryLive,
  useOsqueryHistoryState,
} from "@features/osquery/live/history";

import { CheckForm, emptyCheck } from "./fields";
import { useCreateCheck } from "./queries";

const routeApi = getRouteApi("/_authenticated/osquery/checks/new/");

export function CheckCreatePage() {
  const navigate = routeApi.useNavigate();
  const search = routeApi.useSearch();
  const create = useCreateCheck();
  const historyState = useOsqueryHistoryState();
  const openLive = useOpenOsqueryLive();
  const clearHistoryState = useClearOsqueryHistoryState();
  const draft =
    historyState?.view === "check-form" && historyState.id === undefined
      ? historyState.value
      : undefined;

  return (
    <CheckForm
      initial={emptyCheck}
      draft={draft}
      title="Create Check"
      submitLabel="Create"
      activeTab={search.tab ?? "options"}
      onActiveTabChange={(value) =>
        void navigate({
          search: (previous) => ({
            ...previous,
            tab: value === "targets" ? "targets" : undefined,
          }),
        })
      }
      onCancel={async () => {
        await clearHistoryState();
        await navigate({ to: "/osquery/checks" });
      }}
      onRunLive={(value) =>
        openLive({
          kind: "check",
          sql: value.query.trim(),
          form: {
            view: "check-form",
            value,
          },
        })
      }
      onSubmit={async (value) => (await create.mutateAsync(value)).id}
      onSuccess={async (id) => {
        if (id !== undefined) {
          await clearHistoryState();
          await navigate({
            to: "/osquery/checks/$id",
            params: { id: String(id) },
          });
        }
      }}
    />
  );
}
