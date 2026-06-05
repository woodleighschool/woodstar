import { useMutation } from "@tanstack/react-query";

import type {
  ApiError,
  MunkiArtifact,
  MunkiArtifactMutation,
  MunkiArtifactUpload,
  MunkiArtifactUploadMutation,
} from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";

export type { MunkiArtifact, MunkiArtifactMutation, MunkiArtifactUpload, MunkiArtifactUploadMutation };

export function useCreateMunkiArtifactUpload() {
  return useMutation<MunkiArtifactUpload, ApiError, MunkiArtifactUploadMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/artifact-uploads", { body })),
  });
}

export function useCreateMunkiArtifact() {
  return useMutation<MunkiArtifact, ApiError, MunkiArtifactMutation>({
    mutationFn: (body) => unwrap(apiClient.POST("/api/munki/artifacts", { body })),
  });
}
