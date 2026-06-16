import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useDirectUpload } from "@/hooks/use-direct-upload";
import type { ApiError, MunkiObjectView, MunkiUploadTarget } from "@/lib/api";
import {
  confirmMunkiPackageInstallerUpload,
  confirmMunkiSoftwareIconUpload,
  createMunkiPackageInstallerUpload,
  createMunkiSoftwareIconUpload,
  deleteMunkiPackageInstaller,
  unwrap,
} from "@/lib/api";
import type { UploadTransport } from "@/lib/direct-upload";
import { queryKeys } from "@/lib/query-keys";

type IconUploadVars = { softwareId: number; file: File };
type PackageUploadVars = { packageId: number; file: File };

// useUploadMunkiIcon attaches an icon to existing software.
export function useUploadMunkiIcon() {
  return useDirectUpload<MunkiUploadTarget, MunkiObjectView, IconUploadVars>({
    mutationKey: ["munki-icon-upload"],
    loadingText: "Uploading icon",
    successText: "Icon uploaded",
    errorSurface: "inline",
    createIntent: ({ softwareId, file }) =>
      unwrap(
        createMunkiSoftwareIconUpload({
          path: { id: softwareId },
          body: { filename: file.name, content_type: file.type || undefined },
        }),
      ),
    uploadRequest: uploadRequestFromIntent,
    completeUpload: (intent, { softwareId }) =>
      unwrap(
        confirmMunkiSoftwareIconUpload({
          path: { id: softwareId, object_id: intent.object_id },
        }),
      ),
  });
}

// useUploadMunkiInstaller attaches an installer to an existing package.
export function useUploadMunkiInstaller() {
  return useDirectUpload<MunkiUploadTarget, MunkiObjectView, PackageUploadVars>({
    mutationKey: ["munki-installer-upload"],
    loadingText: "Uploading installer",
    successText: "Installer uploaded",
    errorSurface: "inline",
    createIntent: ({ packageId, file }) =>
      unwrap(
        createMunkiPackageInstallerUpload({
          path: { id: packageId },
          body: { filename: file.name, content_type: file.type || undefined },
        }),
      ),
    uploadRequest: uploadRequestFromIntent,
    completeUpload: (intent, { packageId }) =>
      unwrap(
        confirmMunkiPackageInstallerUpload({
          path: { id: packageId, object_id: intent.object_id },
        }),
      ),
  });
}

export function useDeleteMunkiInstaller() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationKey: ["munki-installer-delete"],
    mutationFn: (packageId) => unwrap(deleteMunkiPackageInstaller({ path: { id: packageId } })),
    onSuccess: (_data, packageId) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackagesAll });
      void queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackage(packageId) });
    },
  });
}

function uploadRequestFromIntent(intent: MunkiUploadTarget) {
  return {
    url: intent.upload_url,
    transport: uploadTransportFromIntent(intent),
    method: intent.method,
    headers: intent.headers ?? {},
  };
}

function uploadTransportFromIntent(intent: MunkiUploadTarget): UploadTransport {
  return intent.upload_kind === "presigned" ? "uppy-s3" : "xhr";
}
