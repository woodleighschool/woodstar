import {
  keepPreviousData,
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import type {
  ApiError,
  OsqueryReport,
  OsqueryReportMutation,
  OsqueryReportSnapshot,
  PageReport,
} from "@lib/api";
import {
  bulkDeleteOsqueryReports,
  createOsqueryReport,
  deleteOsqueryReport,
  getOsqueryReport,
  listOsqueryReportSnapshots,
  listOsqueryReports,
  unwrap,
  updateOsqueryReport,
} from "@lib/api";
import type {
  ListOsqueryReportSnapshotsData,
  ListOsqueryReportsData,
} from "@lib/api-client/types.gen";
import { baseListParams } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

type QueryParams = Record<string, unknown>;

type ReportListParams = NonNullable<ListOsqueryReportsData["query"]>;
type ReportSnapshotParams = NonNullable<ListOsqueryReportSnapshotsData["query"]>;

const REPORT_SNAPSHOT_REFRESH_MS = 30_000;

const reportKeys = {
  all: ["osquery", "reports"] as const,
  list: (params?: QueryParams) => ["osquery", "reports", "list", params ?? {}] as const,
  detail: (id: number | null) => ["osquery", "reports", "detail", id] as const,
  snapshotsRoot: (id: number | null) => ["osquery", "reports", "detail", id, "snapshots"] as const,
  snapshots: (id: number | null, params?: QueryParams) =>
    [...reportKeys.snapshotsRoot(id), params ?? {}] as const,
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

export function useReportSnapshots(id: number | null, params: ReportSnapshotParams = {}) {
  return useQuery<OsqueryReportSnapshot[], ApiError>({
    queryKey: reportKeys.snapshots(id, params),
    queryFn: ({ signal }) =>
      unwrap(
        listOsqueryReportSnapshots({
          path: detailPath(id),
          query: params,
          signal,
        }),
      ),
    enabled: id !== null,
    placeholderData: keepPreviousData,
    refetchInterval: REPORT_SNAPSHOT_REFRESH_MS,
  });
}

export function useCreateReport() {
  const queryClient = useQueryClient();
  return useMutation<OsqueryReport, ApiError, OsqueryReportMutation>({
    mutationFn: (body) => unwrap(createOsqueryReport({ body })),
    onSuccess: async () => {
      toast.add({ title: "Report created", type: "success" });
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
    onSuccess: async (saved) => {
      const previous = queryClient.getQueryData<OsqueryReport>(reportKeys.detail(id));
      const queryChanged = previous !== undefined && previous.query !== saved.query;
      if (queryChanged) {
        queryClient.removeQueries({ queryKey: reportKeys.snapshotsRoot(id) });
      }
      toast.add({
        title: queryChanged ? "Report saved and results cleared" : "Report saved",
        type: "success",
      });
      await queryClient.invalidateQueries({ queryKey: reportKeys.all });
    },
  });
}

export function useDeleteReport() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteOsqueryReport({ path: { id } })),
    onSuccess: async () => {
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
