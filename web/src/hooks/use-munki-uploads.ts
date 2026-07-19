import { useUpload } from "@/hooks/use-upload";
import type {
  MunkiDirectUploadTarget,
  MunkiObjectView,
  MunkiPackageInstallerUploadTarget,
} from "@/lib/api";
import {
  createMunkiPackageInstaller,
  createMunkiSoftwareIconUpload,
  finalizeMunkiPackageInstaller,
  setMunkiSoftwareIcon,
  unwrap,
} from "@/lib/api";
import { deleteUnclaimedMunkiInstaller, uploadRequestFromTarget } from "@/lib/munki-upload";

type IconUploadVars = { softwareId: number; file: File };
type PackageUploadVars = { file: File };

// useUploadMunkiIcon attaches an icon to existing software.
export function useUploadMunkiIcon() {
  return useUpload<MunkiDirectUploadTarget, MunkiObjectView, IconUploadVars>({
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
    uploadRequest: (intent) => uploadRequestFromTarget(intent),
    completeUpload: (intent, { softwareId }, signal) =>
      unwrap(
        setMunkiSoftwareIcon({
          path: { id: softwareId },
          body: { object_id: intent.object_id },
          signal,
        }),
      ),
  });
}

// useUploadMunkiInstaller reserves, uploads, and finalizes an unclaimed installer object.
export function useUploadMunkiInstaller() {
  return useUpload<MunkiPackageInstallerUploadTarget, MunkiObjectView, PackageUploadVars>({
    mutationKey: ["munki-installer-upload"],
    loadingText: "Uploading installer",
    successText: "Installer uploaded",
    createIntent: ({ file }) =>
      unwrap(createMunkiPackageInstaller({ body: { filename: file.name } })),
    uploadRequest: (intent) => uploadRequestFromTarget(intent),
    completeUpload: (intent, _vars, signal) =>
      unwrap(finalizeMunkiPackageInstaller({ path: { id: intent.object_id }, signal })),
    cleanupIntent: (intent) => deleteUnclaimedMunkiInstaller(intent.object_id),
  });
}
