import type { MunkiDirectUploadTarget, MunkiPackageInstallerUploadTarget } from "@/lib/api";
import {
  completeMunkiPackageInstallerMultipart,
  createMunkiPackageInstallerMultipart,
  deleteMunkiPackageInstallerUpload,
  signMunkiPackageInstallerPart,
  unwrap,
} from "@/lib/api";
import type { MultipartUploadRequest, UploadRequest } from "@/lib/upload";
import { assertNever } from "@/lib/utils";

export function uploadRequestFromTarget(
  target: MunkiDirectUploadTarget | MunkiPackageInstallerUploadTarget,
): UploadRequest {
  const upload = target.upload;
  switch (upload.strategy) {
    case "direct-put":
      return {
        strategy: "direct-put",
        url: upload.url,
        method: upload.method,
        headers: upload.headers ?? {},
      };
    case "multipart":
      return {
        strategy: "multipart",
        multipart: packageInstallerMultipartRequest(target.object_id),
      };
  }
  return assertNever(upload);
}

export async function deleteUnclaimedMunkiInstaller(objectID: number) {
  await unwrap(deleteMunkiPackageInstallerUpload({ path: { id: objectID } }));
}

function packageInstallerMultipartRequest(objectID: number): MultipartUploadRequest {
  return {
    createMultipartUpload: async () => {
      const upload = await unwrap(createMunkiPackageInstallerMultipart({ path: { id: objectID } }));
      return { uploadId: upload.upload_id, key: upload.key };
    },
    signPart: async (part) => {
      const target = await unwrap(
        signMunkiPackageInstallerPart({
          path: { id: objectID, part_number: part.partNumber },
          signal: part.signal,
        }),
      );
      return {
        method: target.method,
        url: target.upload_url,
        headers: target.headers ?? {},
      };
    },
    completeMultipartUpload: async (upload) => {
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
    },
    abortMultipartUpload: async () => deleteUnclaimedMunkiInstaller(objectID),
  };
}
