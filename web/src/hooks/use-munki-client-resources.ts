import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { useUpload } from "@/hooks/use-upload";
import type {
  ApiError,
  MunkiBuilder,
  MunkiClientResources,
  MunkiDirectUploadTarget,
} from "@/lib/api";
import {
  createMunkiClientResourcesBannerUpload,
  createMunkiClientResourcesArchiveUpload,
  deleteMunkiClientResources,
  deleteMunkiClientResourcesArchiveUpload,
  deleteMunkiClientResourcesBannerUpload,
  getMunkiClientResources,
  nullOn404,
  publishMunkiClientResourcesArchive,
  updateMunkiClientResourcesBuilder,
  unwrap,
} from "@/lib/api";
import { uploadRequestFromTarget } from "@/lib/munki-upload";
import { queryKeys } from "@/lib/query-keys";

type BannerUploadVariables = {
  file: File;
  body: Omit<MunkiBuilder, "banner_object_id">;
};

type SaveBuilderVariables = {
  body: MunkiBuilder;
  signal: AbortSignal;
};

export function useMunkiClientResources() {
  return useQuery<MunkiClientResources | null, ApiError>({
    queryKey: queryKeys.munkiClientResources,
    queryFn: ({ signal }) => nullOn404(getMunkiClientResources({ signal })),
  });
}

export function useSaveMunkiClientResourcesBuilder() {
  const queryClient = useQueryClient();
  return useMutation<MunkiClientResources, ApiError, SaveBuilderVariables>({
    mutationFn: ({ body, signal }) => unwrap(updateMunkiClientResourcesBuilder({ body, signal })),
    onSuccess: async () => {
      toast.success("Client resources saved");
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiClientResources });
    },
  });
}

export function useUploadAndSaveMunkiClientResourcesBanner() {
  const queryClient = useQueryClient();
  return useUpload<MunkiDirectUploadTarget, MunkiClientResources, BannerUploadVariables>({
    mutationKey: ["munki-client-resources-banner-upload"],
    loadingText: "Saving client resources",
    successText: "Client resources saved",
    createIntent: ({ file }) =>
      unwrap(
        createMunkiClientResourcesBannerUpload({
          body: {
            filename: file.name,
          },
        }),
      ),
    uploadRequest: (intent) => uploadRequestFromTarget(intent),
    completeUpload: (intent, { body }, signal) =>
      unwrap(
        updateMunkiClientResourcesBuilder({
          body: { ...body, banner_object_id: intent.object_id },
          signal,
        }),
      ),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiClientResources });
    },
    cleanupIntent: (intent) =>
      unwrap(deleteMunkiClientResourcesBannerUpload({ path: { id: intent.object_id } })),
  });
}

export function useUploadAndPublishMunkiClientResourcesArchive() {
  const queryClient = useQueryClient();
  return useUpload<MunkiDirectUploadTarget, MunkiClientResources>({
    mutationKey: ["munki-client-resources-archive-upload"],
    loadingText: "Saving client resources",
    successText: "Client resources saved",
    createIntent: ({ file }) =>
      unwrap(createMunkiClientResourcesArchiveUpload({ body: { filename: file.name } })),
    uploadRequest: (intent) => uploadRequestFromTarget(intent),
    completeUpload: (intent, _variables, signal) =>
      unwrap(
        publishMunkiClientResourcesArchive({
          body: { object_id: intent.object_id },
          signal,
        }),
      ),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiClientResources });
    },
    cleanupIntent: (intent) =>
      unwrap(deleteMunkiClientResourcesArchiveUpload({ path: { id: intent.object_id } })),
  });
}

export function useDeleteMunkiClientResources() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError>({
    mutationFn: () => unwrap(deleteMunkiClientResources()),
    onSuccess: async () => {
      toast.success("Client resources undeployed");
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiClientResources });
    },
  });
}
