import { useDirectUpload } from "@/hooks/use-direct-upload";
import type { MunkiArtifact, MunkiArtifactUpload, MunkiArtifactUploadMutation } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { fileSHA256 } from "@/lib/direct-upload";

export type { MunkiArtifact, MunkiArtifactUpload, MunkiArtifactUploadMutation };

export function useUploadMunkiArtifact(kind: MunkiArtifactUploadMutation["kind"]) {
  const label = kind === "icon" ? "icon" : "package";

  return useDirectUpload<MunkiArtifactUpload, MunkiArtifact>({
    mutationKey: ["munki-artifact-upload", kind],
    loadingText: `Uploading ${label}`,
    successText: `${capitalize(label)} uploaded`,
    errorSurface: "inline",
    createIntent: async (file) => {
      const sha256 = await fileSHA256(file);
      return unwrap(
        apiClient.POST("/api/munki/artifact-uploads", {
          body: {
            kind,
            filename: file.name,
            content_type: file.type || undefined,
            size_bytes: file.size,
            sha256,
          },
        }),
      );
    },
    uploadRequest: (upload) => ({
      url: upload.upload_url,
      headers: upload.headers ?? {},
    }),
    completeUpload: (upload) =>
      unwrap(apiClient.POST("/api/munki/artifacts", { body: upload.artifact })),
  });
}

function capitalize(value: string) {
  return `${value.charAt(0).toUpperCase()}${value.slice(1)}`;
}
