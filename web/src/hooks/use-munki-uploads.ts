import { useDirectUpload } from "@/hooks/use-direct-upload";
import type { MunkiObjectView, MunkiUploadTarget } from "@/lib/api";
import {
  completeMunkiPackageInstallerMultipart,
  createMunkiPackageInstaller,
  createMunkiPackageInstallerMultipart,
  createMunkiSoftwareIconUpload,
  deleteMunkiPackageInstaller,
  finalizeMunkiPackageInstaller,
  setMunkiSoftwareIcon,
  signMunkiPackageInstallerPart,
  unwrap,
} from "@/lib/api";
import type { DirectMultipartUploadRequest, UploadTransport } from "@/lib/direct-upload";

type IconUploadVars = { softwareId: number; file: File };
type PackageUploadVars = { file: File };

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
  return useDirectUpload<MunkiUploadTarget, MunkiObjectView, PackageUploadVars>({
    mutationKey: ["munki-installer-upload"],
    loadingText: "Uploading installer",
    successText: "Installer uploaded",
    createIntent: ({ file }) =>
      unwrap(createMunkiPackageInstaller({ body: { filename: file.name } })),
    uploadRequest: (intent) => installerUploadRequest(intent),
    completeUpload: (intent, _vars, signal) =>
      unwrap(finalizeMunkiPackageInstaller({ path: { id: intent.object_id }, signal })),
    cleanupIntent: (intent) => deleteUnclaimedMunkiInstaller(intent.object_id),
  });
}

export async function deleteUnclaimedMunkiInstaller(objectID: number) {
  await unwrap(deleteMunkiPackageInstaller({ path: { id: objectID } }));
}

function uploadRequestFromIntent(intent: MunkiUploadTarget) {
  return {
    url: intent.upload_url,
    transport: uploadTransportFromIntent(intent),
    method: intent.method,
    headers: intent.headers ?? {},
  };
}

function installerUploadRequest(intent: MunkiUploadTarget) {
  const base = uploadRequestFromIntent(intent);
  if (intent.upload_transport !== "s3") {
    return base;
  }
  return {
    ...base,
    multipart: installerMultipartRequest(intent.object_id),
  };
}

function installerMultipartRequest(objectID: number): DirectMultipartUploadRequest {
  return {
    createMultipartUpload: async () => {
      const upload = await unwrap(createMunkiPackageInstallerMultipart({ path: { id: objectID } }));
      return { uploadId: upload.upload_id, key: upload.key };
    },
    signPart: async (_file, part) => {
      const target = await unwrap(
        signMunkiPackageInstallerPart({
          path: { id: objectID, part_number: part.partNumber },
          signal: part.signal,
        }),
      );
      return {
        method: "PUT",
        url: target.upload_url,
        headers: target.headers ?? {},
      };
    },
    completeMultipartUpload: async (_file, upload) => {
      const parts = upload.parts
        .map((part) => {
          if (part.PartNumber === undefined || part.ETag === undefined) {
            throw new Error("Storage did not return a completed multipart part.");
          }
          return { part_number: part.PartNumber, etag: part.ETag };
        })
        .toSorted((left, right) => left.part_number - right.part_number);
      await unwrap(
        completeMunkiPackageInstallerMultipart({
          path: { id: objectID },
          body: { parts },
          signal: upload.signal,
        }),
      );
      return {};
    },
    abortMultipartUpload: async () => deleteUnclaimedMunkiInstaller(objectID),
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
