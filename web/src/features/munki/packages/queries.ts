import {
  keepPreviousData,
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import { useUpload } from "@hooks/use-upload";
import type {
  ApiError,
  MunkiObjectView,
  MunkiPackage,
  MunkiPackageCreateMutation,
  MunkiPackageInstallerUploadTarget,
  MunkiPackageMutation,
  PagePackage,
} from "@lib/api";
import {
  bulkDeleteMunkiPackages,
  completeMunkiPackageInstallerUpload,
  createMunkiPackage,
  createMunkiPackageInstallerUpload,
  getMunkiPackage,
  listMunkiPackages,
  unwrap,
  updateMunkiPackage,
} from "@lib/api";
import type { ListMunkiPackagesData } from "@lib/api-client/types.gen";
import { baseListParams } from "@lib/pagination";
import { detailPath } from "@lib/route-params";

import { deleteUnclaimedMunkiInstaller, uploadRequestFromTarget } from "../upload";

type MunkiPackageListParams = NonNullable<ListMunkiPackagesData["query"]>;
type PackageUploadVariables = { file: File };
type QueryParams = Record<string, unknown>;

const munkiRoot = ["munki"] as const;

const munkiPackageKeys = {
  list: (params: QueryParams) => [...munkiRoot, "packages", "list", params] as const,
  detail: (id: number | null) => [...munkiRoot, "packages", "detail", id] as const,
};

function packageQueryParams(params: MunkiPackageListParams) {
  return {
    ...baseListParams(params),
    type: params.type?.length ? params.type : undefined,
    software_id: params.software_id === 0 ? undefined : params.software_id,
  };
}

export function munkiPackageQueryOptions(id: number | null) {
  return queryOptions<MunkiPackage, ApiError>({
    queryKey: munkiPackageKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getMunkiPackage({ path: detailPath(id), signal })),
    enabled: id !== null,
  });
}

export function useMunkiPackages(params: MunkiPackageListParams = {}) {
  const query = packageQueryParams(params);
  return useQuery<PagePackage, ApiError>({
    queryKey: munkiPackageKeys.list(query),
    queryFn: ({ signal }) => unwrap(listMunkiPackages({ query, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useMunkiPackage(id: number | null) {
  return useQuery(munkiPackageQueryOptions(id));
}

export function useCreateMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<
    MunkiPackage,
    ApiError,
    { body: MunkiPackageCreateMutation; signal?: AbortSignal }
  >({
    mutationFn: ({ body, signal }) => unwrap(createMunkiPackage({ body, signal })),
    onSuccess: async () => {
      toast.add({ title: "Package Created", type: "success" });
      await queryClient.invalidateQueries({ queryKey: munkiRoot });
    },
  });
}

export function useUpdateMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<
    MunkiPackage,
    ApiError,
    { id: number; body: MunkiPackageMutation; signal?: AbortSignal }
  >({
    mutationFn: ({ id, body, signal }) =>
      unwrap(updateMunkiPackage({ path: { id }, body, signal })),
    onSuccess: async () => {
      toast.add({ title: "Package Saved", type: "success" });
      await queryClient.invalidateQueries({ queryKey: munkiRoot });
    },
  });
}

export function useBulkDeleteMunkiPackages() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number[]>({
    mutationFn: (ids) => unwrap(bulkDeleteMunkiPackages({ query: { ids } })),
    onSuccess: async () => queryClient.invalidateQueries({ queryKey: munkiRoot }),
  });
}

export function useDeleteMunkiPackage() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(bulkDeleteMunkiPackages({ query: { ids: [id] } })),
    onSuccess: async () => queryClient.invalidateQueries({ queryKey: munkiRoot }),
  });
}

export function useUploadMunkiInstaller() {
  return useUpload<MunkiPackageInstallerUploadTarget, MunkiObjectView, PackageUploadVariables>({
    mutationKey: ["munki-installer-upload"],
    loadingText: "Uploading Installer",
    successText: "Installer Uploaded",
    createIntent: ({ file }) =>
      unwrap(createMunkiPackageInstallerUpload({ body: { filename: file.name } })),
    uploadRequest: uploadRequestFromTarget,
    completeUpload: (intent, _variables, signal) =>
      unwrap(completeMunkiPackageInstallerUpload({ path: { id: intent.object_id }, signal })),
    cleanupIntent: (intent) => deleteUnclaimedMunkiInstaller(intent.object_id),
  });
}
