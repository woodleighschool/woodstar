import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import {
  useClearOsqueryHistoryState,
  useOpenOsqueryLive,
  useOsqueryHistoryState,
} from "@features/osquery/live/history";
import { parseRouteID } from "@lib/route-params";

import { CheckForm, checkFromDetail } from "./fields";
import { useCheck, useUpdateCheck } from "./queries";

export function CheckEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const checkId = params.id ?? "";
  const id = parseRouteID(checkId);
  const detail = useCheck(id);
  const update = useUpdateCheck(id);
  const historyState = useOsqueryHistoryState();
  const openLive = useOpenOsqueryLive();
  const clearHistoryState = useClearOsqueryHistoryState();

  if (id === null) {
    return (
      <QueryGate title="Failed to load check" error={{ message: "Check route is invalid." }} />
    );
  }

  if (detail.error || !detail.data) {
    return (
      <QueryGate
        title="Failed to load check"
        error={detail.error}
        onRetry={() => void detail.refetch()}
      />
    );
  }

  const check = detail.data;
  const draft =
    historyState?.view === "check-form" && historyState.id === check.id
      ? historyState.value
      : undefined;
  return (
    <CheckForm
      key={check.id}
      initial={checkFromDetail(check)}
      draft={draft}
      title="Edit Check"
      submitLabel="Save"
      confirmResultReset
      onCancel={async () => {
        await clearHistoryState();
        await navigate({
          to: "/osquery/checks/$id",
          params: { id: String(check.id) },
        });
      }}
      onRunLive={(value) =>
        openLive({
          kind: "check",
          id: check.id,
          sql: value.query.trim(),
          form: {
            view: "check-form",
            id: check.id,
            value,
          },
        })
      }
      onSubmit={async (value) => (await update.mutateAsync(value)).id}
      onSuccess={async (savedID) => {
        if (savedID !== undefined) {
          await clearHistoryState();
          await navigate({
            to: "/osquery/checks/$id",
            params: { id: String(savedID) },
          });
        }
      }}
    />
  );
}
