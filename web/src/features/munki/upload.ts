import type { MultipartUploadRequest, UploadRequest } from "@woodleighschool/bloby-client";

import type { MunkiUploadTarget } from "@lib/api";
import {
  completeMunkiPackageInstallerMultipart,
  deleteMunkiPackageInstallerUpload,
  signMunkiPackageInstallerPart,
  unwrap,
} from "@lib/api";
import { assertNever } from "@lib/utils";

export function uploadRequestFromTarget(target: MunkiUploadTarget): UploadRequest {
  const upload = target.upload;
  switch (upload.strategy) {
    case "direct-put":
      return upload;
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
    complete: (parts, signal) =>
      unwrap(
        completeMunkiPackageInstallerMultipart({
          path: { id: objectID },
          body: { parts },
          signal,
        }),
      ),
    signPart: (partNumber, signal) =>
      unwrap(
        signMunkiPackageInstallerPart({
          path: { id: objectID, part_number: partNumber },
          signal,
        }),
      ),
  };
}
