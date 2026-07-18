import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type {
  ApiError,
  OsqueryReport,
  OsqueryReportMutation,
  OsqueryReportResult,
  PageReport,
} from "@/lib/api";
import {
  bulkDeleteOsqueryReports,
  createOsqueryReport,
  deleteOsqueryReport,
  getOsqueryReport,
  listOsqueryReportResults,
  listOsqueryReports,
  unwrap,
  updateOsqueryReport,
} from "@/lib/api";
import type { ListOsqueryReportsData } from "@/lib/api-client/types.gen";
import { baseListParams } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { detailPath } from "@/lib/route-params";

export type ReportListParams = NonNullable<ListOsqueryReportsData["query"]>;

const REPORT_RESULT_REFRESH_MS = 30_000;

export function useReports(params: ReportListParams = {}) {
  const queryParams = baseListParams(params);

  return useQuery<PageReport, ApiError>({
    queryKey: queryKeys.reports(queryParams),
    queryFn: ({ signal }) => unwrap(listOsqueryReports({ query: queryParams, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useReport(id: number | null) {
  return useQuery<OsqueryReport, ApiError>({
    queryKey: queryKeys.report(id),
    queryFn: ({ signal }) =>
      unwrap(
        getOsqueryReport({
          path: detailPath(id),
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useReportResults(id: number | null) {
  return useQuery<OsqueryReportResult[], ApiError>({
    queryKey: queryKeys.reportResults(id),
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
      await queryClient.invalidateQueries({ queryKey: queryKeys.reportsAll });
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
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.reportsAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.report(id) }),
        queryClient.invalidateQueries({ queryKey: queryKeys.reportResults(id) }),
      ]);
    },
  });
}

export function useDeleteReport() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteOsqueryReport({ path: { id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.reportsAll });
    },
  });
}

export function useBulkDeleteReports() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteOsqueryReports({ query: { ids } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.reportsAll });
    },
  });
}
