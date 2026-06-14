import { useDirectUpload } from "@/hooks/use-direct-upload";
import type { MunkiObject, MunkiUploadTarget } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";

type IconUploadVars = { softwareId: number; file: File };
type PackageUploadVars = { packageId: number; file: File };

// useUploadMunkiIcon attaches an icon to existing software.
export function useUploadMunkiIcon() {
  return useDirectUpload<MunkiUploadTarget, MunkiObject, IconUploadVars>({
    mutationKey: ["munki-icon-upload"],
    loadingText: "Uploading icon",
    successText: "Icon uploaded",
    errorSurface: "inline",
    createIntent: ({ softwareId, file }) =>
      unwrap(
        apiClient.POST("/api/munki/software/{id}/icon", {
          params: { path: { id: softwareId } },
          body: { filename: file.name, content_type: file.type || undefined },
        }),
      ),
    uploadRequest: uploadRequestFromIntent,
    completeUpload: (intent, { softwareId }) =>
      unwrap(
        apiClient.POST("/api/munki/software/{id}/icon/{object_id}/confirm", {
          params: { path: { id: softwareId, object_id: intent.object_id } },
        }),
      ),
  });
}

// useUploadMunkiInstaller attaches an installer to an existing package.
export function useUploadMunkiInstaller() {
  return useDirectUpload<MunkiUploadTarget, MunkiObject, PackageUploadVars>({
    mutationKey: ["munki-installer-upload"],
    loadingText: "Uploading installer",
    successText: "Installer uploaded",
    errorSurface: "inline",
    createIntent: ({ packageId, file }) =>
      unwrap(
        apiClient.POST("/api/munki/packages/{id}/installer", {
          params: { path: { id: packageId } },
          body: { filename: file.name, content_type: file.type || undefined },
        }),
      ),
    uploadRequest: uploadRequestFromIntent,
    completeUpload: (intent, { packageId }) =>
      unwrap(
        apiClient.POST("/api/munki/packages/{id}/installer/{object_id}/confirm", {
          params: { path: { id: packageId, object_id: intent.object_id } },
        }),
      ),
  });
}

// useUploadMunkiUninstaller attaches an uninstaller to an existing package.
export function useUploadMunkiUninstaller() {
  return useDirectUpload<MunkiUploadTarget, MunkiObject, PackageUploadVars>({
    mutationKey: ["munki-uninstaller-upload"],
    loadingText: "Uploading uninstaller",
    successText: "Uninstaller uploaded",
    errorSurface: "inline",
    createIntent: ({ packageId, file }) =>
      unwrap(
        apiClient.POST("/api/munki/packages/{id}/uninstaller", {
          params: { path: { id: packageId } },
          body: { filename: file.name, content_type: file.type || undefined },
        }),
      ),
    uploadRequest: uploadRequestFromIntent,
    completeUpload: (intent, { packageId }) =>
      unwrap(
        apiClient.POST("/api/munki/packages/{id}/uninstaller/{object_id}/confirm", {
          params: { path: { id: packageId, object_id: intent.object_id } },
        }),
      ),
  });
}

function uploadRequestFromIntent(intent: MunkiUploadTarget) {
  return {
    url: intent.upload_url,
    method: intent.method,
    headers: intent.headers ?? {},
  };
}
