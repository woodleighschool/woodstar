import type { MunkiUploadTarget } from "@/lib/api";
import type { MultipartUploadRequest, UploadRequest } from "@/lib/upload";

export function uploadRequestFromTarget(
  target: MunkiUploadTarget,
  multipart?: MultipartUploadRequest,
): UploadRequest {
  if (target.upload.strategy === "direct-put") {
    return {
      strategy: "direct-put",
      url: target.upload.url,
      method: target.upload.method,
      headers: target.upload.headers ?? {},
    };
  }
  if (!multipart) {
    throw new Error("Multipart upload support is not configured.");
  }
  return { strategy: "multipart", multipart };
}
