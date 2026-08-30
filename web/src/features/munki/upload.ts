import type { MunkiDirectUploadTarget, MunkiPackageInstallerUploadTarget } from "@lib/api";
import {
  completeMunkiPackageInstallerMultipart,
  deleteMunkiPackageInstallerUpload,
  signMunkiPackageInstallerPart,
  unwrap,
} from "@lib/api";
import type { MultipartUploadRequest, UploadRequest, UploadResult } from "@lib/upload";
import { assertNever } from "@lib/utils";

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

export async function completeMunkiInstallerTransfer(
  objectID: number,
  transfer: UploadResult,
  signal?: AbortSignal,
) {
  if (transfer.strategy === "direct-put") return;

  const request = () =>
    unwrap(
      completeMunkiPackageInstallerMultipart({
        path: { id: objectID },
        body: {
          parts: transfer.parts.map((part) => ({
            part_number: part.partNumber,
            etag: part.etag,
          })),
        },
        signal,
      }),
    );

  try {
    await request();
  } catch (error) {
    if (signal?.aborted) throw error;
    await request();
  }
}

export async function deleteUnclaimedMunkiInstaller(objectID: number) {
  await unwrap(deleteMunkiPackageInstallerUpload({ path: { id: objectID } }));
}

function packageInstallerMultipartRequest(objectID: number): MultipartUploadRequest {
  return {
    signPart: async (partNumber, signal) => {
      const target = await unwrap(
        signMunkiPackageInstallerPart({
          path: { id: objectID, part_number: partNumber },
          signal,
        }),
      );
      return {
        method: target.method,
        url: target.url,
        headers: target.headers ?? {},
      };
    },
  };
}
