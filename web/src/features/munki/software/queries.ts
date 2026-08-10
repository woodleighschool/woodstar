import {
  keepPreviousData,
  type QueryClient,
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import { useUpload } from "@hooks/use-upload";
import type {
  ApiError,
  MunkiCreateMutation,
  MunkiDirectUploadTarget,
  MunkiObjectView,
  MunkiSoftwareDetail,
  MunkiUpdateMutation,
  PageMunkiObjectView,
  PageSoftware,
  PageSoftwareReportHost,
} from "@lib/api";
import {
  bulkDeleteMunkiSoftware,
  createMunkiIconUpload,
  createMunkiSoftware,
  deleteMunkiSoftware,
  getMunkiSoftware,
  listMunkiIcons,
  listMunkiSoftware,
  listMunkiSoftwareReport,
  setMunkiSoftwareIcon,
  unwrap,
  updateMunkiSoftware,
} from "@lib/api";
import type { ListMunkiSoftwareData, ListMunkiSoftwareReportData } from "@lib/api-client/types.gen";
import { baseListParams, MAX_PAGE_SIZE } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

import { uploadRequestFromTarget } from "../upload";

type MunkiListParams = NonNullable<ListMunkiSoftwareData["query"]>;
type MunkiSoftwareReportParams = NonNullable<ListMunkiSoftwareReportData["query"]>;
type IconUploadVariables = { softwareId: number; file: File };
type QueryParams = Record<string, unknown>;

const munkiRoot = ["munki"] as const;

const munkiSoftwareKeys = {
  root: [...munkiRoot, "software"] as const,
  list: (params: QueryParams) => [...munkiRoot, "software", "list", params] as const,
  detail: (id: number | null) => [...munkiRoot, "software", "detail", id] as const,
  report: (id: number | null, params: QueryParams) =>
    [...munkiRoot, "software", "detail", id, "report", params] as const,
  iconList: (params: QueryParams) => [...munkiRoot, "icons", "list", params] as const,
};

async function invalidateMunkiCatalog(queryClient: QueryClient) {
  await queryClient.invalidateQueries({ queryKey: munkiRoot });
}

export function munkiSoftwareQueryOptions(id: number | null) {
  return queryOptions<MunkiSoftwareDetail, ApiError>({
    queryKey: munkiSoftwareKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getMunkiSoftware({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function useMunkiSoftware(params: MunkiListParams = {}) {
  const query = baseListParams(params);
  return useQuery<PageSoftware, ApiError>({
    queryKey: munkiSoftwareKeys.list(query),
    queryFn: ({ signal }) => unwrap(listMunkiSoftware({ query, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiSoftwareDetail(id: number | null) {
  return useQuery(munkiSoftwareQueryOptions(id));
}

export function useMunkiSoftwareReport(id: number | null, params: MunkiSoftwareReportParams = {}) {
  const query = { ...baseListParams(params), status: params.status };
  return useQuery<PageSoftwareReportHost, ApiError>({
    queryKey: munkiSoftwareKeys.report(id, query),
    queryFn: ({ signal }) =>
      unwrap(listMunkiSoftwareReport({ path: detailPath(id), query, signal })),
    enabled: id !== null,
    placeholderData: keepPreviousData,
  });
}

export function useCreateMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareDetail, ApiError, MunkiCreateMutation>({
    mutationFn: (body) => unwrap(createMunkiSoftware({ body })),
    onSuccess: async () => {
      toast.add({ title: "Software created", type: "success" });
      await queryClient.invalidateQueries({ queryKey: munkiSoftwareKeys.root });
    },
  });
}

export function useUpdateMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<MunkiSoftwareDetail, ApiError, { id: number; body: MunkiUpdateMutation }>({
    mutationFn: ({ id, body }) => unwrap(updateMunkiSoftware({ path: { id }, body })),
    onSuccess: async () => {
      toast.add({ title: "Software saved", type: "success" });
      await invalidateMunkiCatalog(queryClient);
    },
  });
}

export function useBulkDeleteMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteMunkiSoftware({ query: { ids } })),
    onSuccess: async () => invalidateMunkiCatalog(queryClient),
  });
}

export function useDeleteMunkiSoftware() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteMunkiSoftware({ path: { id } })),
    onSuccess: async () => invalidateMunkiCatalog(queryClient),
  });
}

export function useMunkiIcons(enabled = true) {
  const query = baseListParams({}, { defaultPerPage: MAX_PAGE_SIZE });
  return useQuery<PageMunkiObjectView, ApiError>({
    queryKey: munkiSoftwareKeys.iconList(query),
    queryFn: ({ signal }) => unwrap(listMunkiIcons({ query, signal })),
    enabled,
  });
}

export function useUploadMunkiIcon() {
  const queryClient = useQueryClient();
  return useUpload<MunkiDirectUploadTarget, MunkiObjectView, IconUploadVariables>({
    mutationKey: ["munki-icon-upload"],
    loadingText: "Uploading icon",
    successText: "Icon uploaded",
    errorSurface: "inline",
    createIntent: ({ file }) => unwrap(createMunkiIconUpload({ body: { filename: file.name } })),
    uploadRequest: uploadRequestFromTarget,
    completeUpload: (intent, { softwareId }, signal) =>
      unwrap(
        setMunkiSoftwareIcon({
          path: { id: softwareId },
          body: { object_id: intent.object_id },
          signal,
        }),
      ),
    onSuccess: async () => invalidateMunkiCatalog(queryClient),
  });
}
