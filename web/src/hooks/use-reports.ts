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
import { DEFAULT_PAGE_SIZE } from "@/lib/pagination";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { OsqueryReport, OsqueryReportMutation };
export type ReportListResult = PageReport;
export type ReportResults = OsqueryReportResult[];
export type ReportListParams = NonNullable<ListOsqueryReportsData["query"]>;

export function useReports(params: ReportListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
  };

  return useQuery<ReportListResult, ApiError>({
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
          path: { id: id ?? 0 },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useReportResults(id: number | null) {
  return useQuery<ReportResults, ApiError>({
    queryKey: queryKeys.reportResults(id),
    queryFn: ({ signal }) =>
      unwrap(
        listOsqueryReportResults({
          path: { id: id ?? 0 },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCreateReport() {
  const queryClient = useQueryClient();
  return useMutation<OsqueryReport, ApiError, OsqueryReportMutation>({
    mutationFn: (body) => unwrap(createOsqueryReport({ body })),
    onSuccess: () => {
      toast.success("Report created");
      void queryClient.invalidateQueries({ queryKey: queryKeys.reportsAll });
    },
  });
}

export function useUpdateReport(id: number | null) {
  const queryClient = useQueryClient();
  return useMutation<OsqueryReport, ApiError, OsqueryReportMutation>({
    mutationFn: (body) =>
      unwrap(
        updateOsqueryReport({
          path: { id: id ?? 0 },
          body,
        }),
      ),
    onSuccess: () => {
      toast.success("Report saved");
      void queryClient.invalidateQueries({ queryKey: queryKeys.reportsAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.report(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.reportResults(id) });
    },
  });
}

export function useDeleteReport() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteOsqueryReport({ path: { id } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.reportsAll });
    },
  });
}

export function useBulkDeleteReports() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteOsqueryReports({ body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.reportsAll });
    },
  });
}
