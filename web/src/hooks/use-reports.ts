import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { DEFAULT_PAGE_SIZE } from "@/hooks/use-data-table-search";
import type { ApiError, Page, Report, ReportMutation, ReportResult } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { nonEmpty } from "@/lib/utils";

export type { Report, ReportMutation };
export type ReportListResult = Page<Report>;
export type ReportResults = ReportResult[];

export interface ReportListParams {
  q?: string;
  page?: number;
  per_page?: number;
  sort?: string;
}

export function useReports(params: ReportListParams = {}) {
  const queryParams = {
    q: nonEmpty(params.q),
    page: params.page ?? 1,
    per_page: params.per_page ?? DEFAULT_PAGE_SIZE,
    sort: nonEmpty(params.sort),
  };

  return useQuery<ReportListResult, ApiError>({
    queryKey: queryKeys.reports(queryParams),
    queryFn: ({ signal }) =>
      unwrap(apiClient.GET("/api/osquery/reports", { params: { query: queryParams }, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useReport(id: number | null) {
  return useQuery<Report, ApiError>({
    queryKey: queryKeys.report(id),
    queryFn: ({ signal }) =>
      unwrap(
        apiClient.GET("/api/osquery/reports/{id}", {
          params: { path: { id: id ?? 0 } },
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
        apiClient.GET("/api/osquery/reports/{id}/results", {
          params: { path: { id: id ?? 0 } },
          signal,
        }),
      ),
    enabled: id !== null,
  });
}

export function useCreateReport() {
  const queryClient = useQueryClient();
  return useMutation<Report, ApiError, ReportMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/osquery/reports", { body })),
    onSuccess: () => {
      toast.success("Report created");
      void queryClient.invalidateQueries({ queryKey: queryKeys.reportsAll });
    },
  });
}

export function useUpdateReport(id: number | null) {
  const queryClient = useQueryClient();
  return useMutation<Report, ApiError, ReportMutation>({
    mutationFn: (body) =>
      unwrap(
        apiClient.PUT("/api/osquery/reports/{id}", {
          params: { path: { id: id ?? 0 } },
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
    mutationFn: (id) =>
      unwrap(apiClient.DELETE("/api/osquery/reports/{id}", { params: { path: { id } } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.reportsAll });
    },
  });
}

export function useBulkDeleteReports() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) =>
      unwrap(apiClient.POST("/api/osquery/reports/bulk-delete", { body: { ids } })),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.reportsAll });
    },
  });
}
