import { useMutation, useQueryClient } from "@tanstack/react-query";

import { useDirectUpload } from "@/hooks/use-direct-upload";
import type { ApiError, MunkiObjectView, MunkiUploadTarget } from "@/lib/api";
import {
  createMunkiPackageInstallerUpload,
  createMunkiSoftwareIconUpload,
  deleteMunkiPackageInstaller,
  setMunkiPackageInstaller,
  setMunkiSoftwareIcon,
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
          body: { filename: file.name },
        }),
      ),
    uploadRequest: uploadRequestFromIntent,
    completeUpload: (intent, { softwareId }) =>
      unwrap(
        setMunkiSoftwareIcon({
          path: { id: softwareId },
          body: { object_id: intent.object_id },
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
          body: { filename: file.name },
        }),
      ),
    uploadRequest: uploadRequestFromIntent,
    completeUpload: (intent, { packageId }) =>
      unwrap(
        setMunkiPackageInstaller({
          path: { id: packageId },
          body: { object_id: intent.object_id },
        }),
      ),
  });
}

export function useDeleteMunkiInstaller() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationKey: ["munki-installer-delete"],
    mutationFn: (packageId) => unwrap(deleteMunkiPackageInstaller({ path: { id: packageId } })),
    onSuccess: async (_data, packageId) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackagesAll }),
        queryClient.invalidateQueries({ queryKey: queryKeys.munkiPackage(packageId) }),
      ]);
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
  switch (intent.upload_transport) {
    case "s3":
      return "uppy-s3";
    case "woodstar":
      return "xhr";
  }
}
