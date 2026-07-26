import {
  keepPreviousData,
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { toast } from "sonner";

import type {
  ApiError,
  OsqueryReport,
  OsqueryReportMutation,
  OsqueryReportResult,
  PageReport,
} from "@lib/api";
import {
  bulkDeleteOsqueryReports,
  createOsqueryReport,
  getOsqueryReport,
  listOsqueryReportResults,
  listOsqueryReports,
  unwrap,
  updateOsqueryReport,
} from "@lib/api";
import type { ListOsqueryReportsData } from "@lib/api-client/types.gen";
import { baseListParams } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

type QueryParams = Record<string, unknown>;

type ReportListParams = NonNullable<ListOsqueryReportsData["query"]>;

const REPORT_RESULT_REFRESH_MS = 30_000;

const reportKeys = {
  all: ["osquery", "reports"] as const,
  list: (params?: QueryParams) => ["osquery", "reports", "list", params ?? {}] as const,
  detail: (id: number | null) => ["osquery", "reports", "detail", id] as const,
  results: (id: number | null) => ["osquery", "reports", "detail", id, "results"] as const,
};

export function reportQueryOptions(id: number | null) {
  return queryOptions<OsqueryReport, ApiError>({
    queryKey: reportKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getOsqueryReport({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function useReports(params: ReportListParams = {}) {
  const queryParams = baseListParams(params);

  return useQuery<PageReport, ApiError>({
    queryKey: reportKeys.list(queryParams),
    queryFn: ({ signal }) => unwrap(listOsqueryReports({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useReport(id: number | null) {
  return useQuery(reportQueryOptions(id));
}

export function useReportResults(id: number | null) {
  return useQuery<OsqueryReportResult[], ApiError>({
    queryKey: reportKeys.results(id),
    queryFn: ({ signal }) =>
      unwrap(
        listOsqueryReportResults({
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
    placeholderData: keepPreviousData,
    refetchInterval: REPORT_RESULT_REFRESH_MS,
  });
}

export function useCreateReport() {
  const queryClient = useQueryClient();
  return useMutation<OsqueryReport, ApiError, OsqueryReportMutation>({
    mutationFn: (body) => unwrap(createOsqueryReport({ body })),
    onSuccess: async () => {
      toast.success("Report created");
      await queryClient.invalidateQueries({ queryKey: reportKeys.all });
    },
  });
}

export function useUpdateReport(id: number | null) {
  const queryClient = useQueryClient();
  return useMutation<OsqueryReport, ApiError, OsqueryReportMutation>({
    mutationFn: (body) =>
      unwrap(
        updateOsqueryReport({
          path: detailPath(id),
          body,
        }),
      ),
    onSuccess: async () => {
      toast.success("Report saved");
      await queryClient.invalidateQueries({ queryKey: reportKeys.all });
    },
  });
}

export function useBulkDeleteReports() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteOsqueryReports({ query: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: reportKeys.all });
    },
  });
}
