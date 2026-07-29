import { useNavigate } from "@tanstack/react-router";

import {
  useClearOsqueryHistoryState,
  useOpenOsqueryLive,
  useOsqueryHistoryState,
} from "@features/osquery/live/history";

import { emptyReport, ReportForm } from "./fields";
import { useCreateReport } from "./queries";

export function ReportCreatePage() {
  const navigate = useNavigate();
  const create = useCreateReport();
  const historyState = useOsqueryHistoryState();
  const openLive = useOpenOsqueryLive();
  const clearHistoryState = useClearOsqueryHistoryState();
  const draft =
    historyState?.view === "report-form" && historyState.id === undefined
      ? historyState.value
      : undefined;

  return (
    <ReportForm
      initial={emptyReport}
      draft={draft}
      title="Create Report"
      submitLabel="Create"
      onCancel={async () => {
        await clearHistoryState();
        await navigate({ to: "/osquery/reports" });
      }}
      onRunLive={(value) =>
        openLive({
          kind: "report",
          sql: value.query.trim(),
          form: {
            view: "report-form",
            value,
          },
        })
      }
      onSubmit={async (value) => (await create.mutateAsync(value)).id}
      onSuccess={async (id) => {
        if (id !== undefined) {
          await clearHistoryState();
          await navigate({
            to: "/osquery/reports/$id",
            params: { id: String(id) },
          });
        }
      }}
    />
  );
}
