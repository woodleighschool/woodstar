import { getRouteApi, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import {
  useClearOsqueryHistoryState,
  useOpenOsqueryLive,
  useOsqueryHistoryState,
} from "@features/osquery/live/history";
import { parseRouteID } from "@lib/route-params";

import { ReportForm, reportFromDetail } from "./fields";
import { useReport, useUpdateReport } from "./queries";

const routeApi = getRouteApi("/_authenticated/osquery/reports/$id/edit");

export function ReportEditPage() {
  const navigate = routeApi.useNavigate();
  const search = routeApi.useSearch();
  const params = useParams({ strict: false });
  const reportId = params.id ?? "";
  const id = parseRouteID(reportId);
  const detail = useReport(id);
  const update = useUpdateReport(id);
  const historyState = useOsqueryHistoryState();
  const openLive = useOpenOsqueryLive();
  const clearHistoryState = useClearOsqueryHistoryState();

  if (id === null) {
    return (
      <QueryGate title="Failed to load report" error={{ message: "Report route is invalid." }} />
    );
  }

  if (detail.error || !detail.data) {
    return (
      <QueryGate
        title="Failed to load report"
        error={detail.error}
        onRetry={() => void detail.refetch()}
      />
    );
  }

  const report = detail.data;
  const draft =
    historyState?.view === "report-form" && historyState.id === report.id
      ? historyState.value
      : undefined;
  return (
    <ReportForm
      key={report.id}
      initial={reportFromDetail(report)}
      draft={draft}
      title="Edit Report"
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
      confirmResultReset
      onCancel={async () => {
        await clearHistoryState();
        await navigate({
          to: "/osquery/reports/$id",
          params: { id: String(report.id) },
        });
      }}
      onRunLive={(value) =>
        openLive({
          kind: "report",
          id: report.id,
          sql: value.query.trim(),
          form: {
            view: "report-form",
            id: report.id,
            value,
          },
        })
      }
      onSubmit={async (value) => (await update.mutateAsync(value)).id}
      onSuccess={async (savedID) => {
        if (savedID !== undefined) {
          await clearHistoryState();
          await navigate({
            to: "/osquery/reports/$id",
            params: { id: String(savedID) },
          });
        }
      }}
    />
  );
}
