import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import { useUpload } from "@hooks/use-upload";
import type {
  ApiError,
  MunkiBuilder,
  MunkiClientResources,
  MunkiClientResourcesMutation,
  MunkiDirectUploadTarget,
  PageMunkiClientResources,
} from "@lib/api";
import {
  createMunkiClientResources,
  createMunkiClientResourcesArchiveUpload,
  createMunkiClientResourcesBannerUpload,
  deleteMunkiClientResources,
  deleteMunkiClientResourcesArchiveUpload,
  deleteMunkiClientResourcesBannerUpload,
  listMunkiClientResources,
  unwrap,
  updateMunkiClientResources,
} from "@lib/api";

import { uploadRequestFromTarget } from "../upload";

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

const munkiClientResourceKeys = {
  root: ["munki", "client-resources"] as const,
};

export function useMunkiClientResources() {
  return useQuery<PageMunkiClientResources, ApiError>({
    queryKey: munkiClientResourceKeys.root,
    queryFn: ({ signal }) => unwrap(listMunkiClientResources({ signal })),
  });
}

export function useSaveMunkiClientResources() {
  const queryClient = useQueryClient();
  return useMutation<MunkiClientResources, ApiError, SaveVariables>({
    mutationFn: saveClientResources,
    onSuccess: async () => {
      toast.add({ title: "Client Resources Saved", type: "success" });
      await queryClient.invalidateQueries({ queryKey: munkiClientResourceKeys.root });
    },
  });
}

export function useUploadAndSaveMunkiClientResourcesBanner() {
  const queryClient = useQueryClient();
  return useUpload<MunkiDirectUploadTarget, MunkiClientResources, BannerUploadVariables>({
    mutationKey: ["munki-client-resources-banner-upload"],
    loadingText: "Saving Client Resources",
    successText: "Client Resources Saved",
    createIntent: ({ file }) =>
      unwrap(createMunkiClientResourcesBannerUpload({ body: { filename: file.name } })),
    uploadRequest: uploadRequestFromTarget,
    completeUpload: (intent, { body, clientResourcesID }, signal) =>
      saveClientResources({
        clientResourcesID,
        body: { builder: { ...body, banner_object_id: intent.object_id } },
        signal,
      }),
    onSuccess: async () =>
      queryClient.invalidateQueries({ queryKey: munkiClientResourceKeys.root }),
    cleanupIntent: (intent) =>
      unwrap(deleteMunkiClientResourcesBannerUpload({ path: { id: intent.object_id } })),
  });
}

export function useUploadAndSaveMunkiClientResourcesArchive() {
  const queryClient = useQueryClient();
  return useUpload<MunkiDirectUploadTarget, MunkiClientResources, ArchiveUploadVariables>({
    mutationKey: ["munki-client-resources-archive-upload"],
    loadingText: "Saving Client Resources",
    successText: "Client Resources Saved",
    createIntent: ({ file }) =>
      unwrap(createMunkiClientResourcesArchiveUpload({ body: { filename: file.name } })),
    uploadRequest: uploadRequestFromTarget,
    completeUpload: (intent, { clientResourcesID }, signal) =>
      saveClientResources({
        clientResourcesID,
        body: { archive_object_id: intent.object_id },
        signal,
      }),
    onSuccess: async () =>
      queryClient.invalidateQueries({ queryKey: munkiClientResourceKeys.root }),
    cleanupIntent: (intent) =>
      unwrap(deleteMunkiClientResourcesArchiveUpload({ path: { id: intent.object_id } })),
  });
}

export function useDeleteMunkiClientResources() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteMunkiClientResources({ path: { id } })),
    onSuccess: async () => {
      toast.add({ title: "Client Resources Undeployed", type: "success" });
      await queryClient.invalidateQueries({ queryKey: munkiClientResourceKeys.root });
    },
  });
}

function saveClientResources({ clientResourcesID, body, signal }: SaveVariables) {
  if (clientResourcesID === null) {
    return unwrap(createMunkiClientResources({ body, signal }));
  }
  return unwrap(updateMunkiClientResources({ path: { id: clientResourcesID }, body, signal }));
}
