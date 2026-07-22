import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { useUpload } from "@/hooks/use-upload";
import type {
  ApiError,
  MunkiBuilder,
  MunkiClientResources,
  MunkiClientResourcesMutation,
  MunkiDirectUploadTarget,
  PageMunkiClientResources,
} from "@/lib/api";
import {
  createMunkiClientResources,
  createMunkiClientResourcesBannerUpload,
  createMunkiClientResourcesArchiveUpload,
  deleteMunkiClientResources,
  deleteMunkiClientResourcesArchiveUpload,
  deleteMunkiClientResourcesBannerUpload,
  listMunkiClientResources,
  updateMunkiClientResources,
  unwrap,
} from "@/lib/api";
import { uploadRequestFromTarget } from "@/lib/munki-upload";
import { queryKeys } from "@/lib/query-keys";

type BannerUploadVariables = {
  file: File;
  clientResourcesID: number | null;
  body: Omit<MunkiBuilder, "banner_object_id">;
};

type ArchiveUploadVariables = {
  file: File;
  clientResourcesID: number | null;
};

type SaveVariables = {
  clientResourcesID: number | null;
  body: MunkiClientResourcesMutation;
  signal: AbortSignal;
};

export function useMunkiClientResources() {
  return useQuery<PageMunkiClientResources, ApiError>({
    queryKey: queryKeys.munkiClientResources,
    queryFn: ({ signal }) => unwrap(listMunkiClientResources({ signal })),
  });
}

export function useSaveMunkiClientResources() {
  const queryClient = useQueryClient();
  return useMutation<MunkiClientResources, ApiError, SaveVariables>({
    mutationFn: saveClientResources,
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
    completeUpload: (intent, { body, clientResourcesID }, signal) =>
      saveClientResources({
        clientResourcesID,
        body: { builder: { ...body, banner_object_id: intent.object_id } },
        signal,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiClientResources });
    },
    cleanupIntent: (intent) =>
      unwrap(deleteMunkiClientResourcesBannerUpload({ path: { id: intent.object_id } })),
  });
}

export function useUploadAndSaveMunkiClientResourcesArchive() {
  const queryClient = useQueryClient();
  return useUpload<MunkiDirectUploadTarget, MunkiClientResources, ArchiveUploadVariables>({
    mutationKey: ["munki-client-resources-archive-upload"],
    loadingText: "Saving client resources",
    successText: "Client resources saved",
    createIntent: ({ file }) =>
      unwrap(createMunkiClientResourcesArchiveUpload({ body: { filename: file.name } })),
    uploadRequest: (intent) => uploadRequestFromTarget(intent),
    completeUpload: (intent, { clientResourcesID }, signal) =>
      saveClientResources({
        clientResourcesID,
        body: { archive_object_id: intent.object_id },
        signal,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiClientResources });
    },
    cleanupIntent: (intent) =>
      unwrap(deleteMunkiClientResourcesArchiveUpload({ path: { id: intent.object_id } })),
  });
}

export function useDeleteMunkiClientResources() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteMunkiClientResources({ path: { id } })),
    onSuccess: async () => {
      toast.success("Client resources undeployed");
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiClientResources });
    },
  });
}

function saveClientResources({ clientResourcesID, body, signal }: SaveVariables) {
  if (clientResourcesID === null) {
    return unwrap(createMunkiClientResources({ body, signal }));
  }
  return unwrap(updateMunkiClientResources({ path: { id: clientResourcesID }, body, signal }));
}
