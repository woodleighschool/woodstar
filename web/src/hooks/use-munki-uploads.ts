import { useDirectUpload } from "@/hooks/use-direct-upload";
import type { MunkiObjectView, MunkiUploadTarget } from "@/lib/api";
import {
  confirmMunkiPackageInstallerUpload,
  confirmMunkiPackageUninstallerUpload,
  confirmMunkiSoftwareIconUpload,
  createMunkiPackageInstallerUpload,
  createMunkiPackageUninstallerUpload,
  createMunkiSoftwareIconUpload,
  unwrap,
} from "@/lib/api";
import type { UploadTransport } from "@/lib/direct-upload";

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

// useUploadMunkiUninstaller attaches an uninstaller to an existing package.
export function useUploadMunkiUninstaller() {
  return useDirectUpload<MunkiUploadTarget, MunkiObjectView, PackageUploadVars>({
    mutationKey: ["munki-uninstaller-upload"],
    loadingText: "Uploading uninstaller",
    successText: "Uninstaller uploaded",
    errorSurface: "inline",
    createIntent: ({ packageId, file }) =>
      unwrap(
        createMunkiPackageUninstallerUpload({
          path: { id: packageId },
          body: { filename: file.name, content_type: file.type || undefined },
        }),
      ),
    uploadRequest: uploadRequestFromIntent,
    completeUpload: (intent, { packageId }) =>
      unwrap(
        confirmMunkiPackageUninstallerUpload({
          path: { id: packageId, object_id: intent.object_id },
        }),
      ),
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
