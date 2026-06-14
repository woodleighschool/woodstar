import { useDirectUpload } from "@/hooks/use-direct-upload";
import type { MunkiUploadTarget, StorageObject } from "@/lib/api";
import { apiClient, unwrap } from "@/lib/api";
import { fileSHA256 } from "@/lib/direct-upload";

type IconUploadVars = { softwareId: number; file: File };
type PackageUploadVars = { packageId: number; file: File };

// useUploadMunkiIcon attaches an icon to existing software: create a pending
// object, upload to the presigned URL, then confirm.
export function useUploadMunkiIcon() {
  return useDirectUpload<MunkiUploadTarget, StorageObject, IconUploadVars>({
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
    uploadRequest: (intent) => ({ url: intent.upload_url, headers: intent.headers ?? {} }),
    completeUpload: (intent, { file }) => confirmObject(intent.object_id, file),
  });
}

// useUploadMunkiInstaller attaches an installer to an existing package.
export function useUploadMunkiInstaller() {
  return useDirectUpload<MunkiUploadTarget, StorageObject, PackageUploadVars>({
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
    uploadRequest: (intent) => ({ url: intent.upload_url, headers: intent.headers ?? {} }),
    completeUpload: (intent, { file }) => confirmObject(intent.object_id, file),
  });
}

// useUploadMunkiUninstaller attaches an uninstaller to an existing package.
export function useUploadMunkiUninstaller() {
  return useDirectUpload<MunkiUploadTarget, StorageObject, PackageUploadVars>({
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
    uploadRequest: (intent) => ({ url: intent.upload_url, headers: intent.headers ?? {} }),
    completeUpload: (intent, { file }) => confirmObject(intent.object_id, file),
  });
}

async function confirmObject(objectId: number, file: File): Promise<StorageObject> {
  const sha256 = await fileSHA256(file);
  return unwrap(
    apiClient.POST("/api/storage/objects/{id}/confirm", {
      params: { path: { id: objectId } },
      body: { sha256 },
    }),
  );
}
