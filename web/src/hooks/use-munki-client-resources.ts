import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { useDirectUpload } from "@/hooks/use-direct-upload";
import type { ApiError, MunkiClientResources, MunkiMutation, MunkiUploadTarget } from "@/lib/api";
import {
  createMunkiClientResourcesBannerUpload,
  deleteMunkiClientResources,
  deleteMunkiClientResourcesBannerUpload,
  getMunkiClientResources,
  nullOn404,
  saveMunkiClientResources,
  unwrap,
} from "@/lib/api";
import type { UploadTransport } from "@/lib/direct-upload";
import { queryKeys } from "@/lib/query-keys";

type BannerUploadVariables = {
  file: File;
  body: Omit<MunkiMutation, "banner_object_id">;
};

export function useMunkiClientResources() {
  return useQuery<MunkiClientResources | null, ApiError>({
    queryKey: queryKeys.munkiClientResources,
    queryFn: ({ signal }) => nullOn404(getMunkiClientResources({ signal })),
  });
}

export function useSaveMunkiClientResources() {
  const queryClient = useQueryClient();
  return useMutation<MunkiClientResources, ApiError, MunkiMutation>({
    mutationFn: (body) => unwrap(saveMunkiClientResources({ body })),
    onSuccess: async () => {
      toast.success("Client resources saved");
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiClientResources });
    },
  });
}

export function useUploadAndSaveMunkiClientResourcesBanner() {
  const queryClient = useQueryClient();
  return useDirectUpload<MunkiUploadTarget, MunkiClientResources, BannerUploadVariables>({
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
    uploadRequest: (intent) => ({
      url: intent.upload_url,
      transport: uploadTransport(intent),
      method: intent.method,
      headers: intent.headers ?? {},
    }),
    completeUpload: async (intent, { body }) => {
      const resource = await unwrap(
        saveMunkiClientResources({
          body: { ...body, banner_object_id: intent.object_id },
        }),
      );
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiClientResources });
      return resource;
    },
    cleanupIntent: (intent) =>
      unwrap(deleteMunkiClientResourcesBannerUpload({ path: { id: intent.object_id } })),
  });
}

export function useDeleteMunkiClientResources() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError>({
    mutationFn: () => unwrap(deleteMunkiClientResources()),
    onSuccess: async () => {
      toast.success("Munki defaults restored");
      await queryClient.invalidateQueries({ queryKey: queryKeys.munkiClientResources });
    },
  });
}

function uploadTransport(intent: MunkiUploadTarget): UploadTransport {
  switch (intent.upload_transport) {
    case "s3":
      return "uppy-s3";
    case "woodstar":
      return "xhr";
  }
}
