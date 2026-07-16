import AwsS3, {
  type AwsS3Options,
  type AwsS3Part,
  type AwsS3UploadParameters,
  type MultipartUploadResultWithSignal,
  type UploadResult,
  type UploadResultWithSignal,
} from "@uppy/aws-s3";
import Uppy, { type Body, type Meta } from "@uppy/core";

export interface UploadProgress {
  loaded: number;
  total: number;
  percent: number;
}

type UploadMeta = Meta;
type UploadBody = Body;
type UploadMethod = "PUT";

export type UploadTransport = "uppy-s3" | "xhr";

export type DirectMultipartUploadResult = UploadResult;

export interface DirectMultipartUploadRequest {
  createMultipartUpload: (file: File) => Promise<DirectMultipartUploadResult>;
  listParts?: (file: File, upload: UploadResultWithSignal) => Promise<AwsS3Part[]>;
  signPart: (
    file: File,
    part: { uploadId: string; key: string; partNumber: number; body: Blob; signal?: AbortSignal },
  ) => Promise<AwsS3UploadParameters>;
  completeMultipartUpload: (
    file: File,
    upload: MultipartUploadResultWithSignal,
  ) => Promise<{ location?: string } | void>;
  abortMultipartUpload: (file: File, upload: UploadResultWithSignal) => Promise<void>;
}

export interface DirectUploadRequest {
  url: string;
  file: File;
  transport: UploadTransport;
  method?: string;
  headers?: Record<string, string>;
  multipart?: DirectMultipartUploadRequest;
  signal?: AbortSignal;
  onProgress?: (progress: UploadProgress) => void;
}

export function uploadWithProgress(request: DirectUploadRequest) {
  switch (request.transport) {
    case "xhr":
      return uploadWithXHRProgress(request);
    case "uppy-s3":
      return uploadWithUppyS3(request);
  }
}

function uploadWithXHRProgress({
  url,
  file,
  method = "PUT",
  headers = {},
  signal,
  onProgress,
}: DirectUploadRequest) {
  return new Promise<void>((resolve, reject) => {
    if (signal?.aborted) {
      reject(new Error("Upload cancelled."));
      return;
    }

    const xhr = new XMLHttpRequest();
    const finish = () => {
      signal?.removeEventListener("abort", abort);
    };
    const abort = () => xhr.abort();

    xhr.upload.onprogress = (event) => {
      const total = event.lengthComputable ? event.total : file.size;
      const percent =
        event.lengthComputable && total > 0 ? Math.round((event.loaded / total) * 100) : 0;
      onProgress?.({ loaded: event.loaded, total, percent });
    };

    xhr.onload = () => {
      finish();
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
        return;
      }
      reject(new Error(`Upload failed with HTTP ${xhr.status}.`));
    };

    xhr.onerror = () => {
      finish();
      reject(new Error("Upload failed before the storage service accepted the request."));
    };

    xhr.onabort = () => {
      finish();
      reject(new Error("Upload cancelled."));
    };

    signal?.addEventListener("abort", abort, { once: true });
    xhr.open(method, url);
    for (const [key, value] of Object.entries(headers)) {
      xhr.setRequestHeader(key, value);
    }
    xhr.send(file);
  });
}

async function uploadWithUppyS3(request: DirectUploadRequest) {
  const { file, signal, onProgress } = request;

  if (signal?.aborted) {
    throw cancelledError();
  }

  const uppy = new Uppy<UploadMeta, UploadBody>({
    autoProceed: false,
    allowMultipleUploadBatches: false,
    restrictions: { maxNumberOfFiles: 1 },
  });
  const abort = () => uppy.cancelAll();

  try {
    signal?.addEventListener("abort", abort, { once: true });
    uppy.use(AwsS3, awsS3Options(request));
    uppy.on("upload-progress", (_uppyFile, progress) => {
      onProgress?.(uploadProgress(file, progress.bytesUploaded, progress.bytesTotal));
    });
    uppy.addFile({ name: file.name, type: file.type, data: file, source: "woodstar" });

    const result = await uppy.upload();
    if (signal?.aborted) {
      throw cancelledError();
    }
    if (!result) {
      throw new Error("Upload did not start.");
    }
    if (result.failed?.length) {
      throw new Error(result.failed[0]?.error ?? "Upload failed.");
    }
    if (!result.successful?.length) {
      throw new Error("Upload did not finish.");
    }
  } catch (error) {
    if (signal?.aborted || isAbortError(error)) {
      throw cancelledError();
    }
    throw error;
  } finally {
    signal?.removeEventListener("abort", abort);
    uppy.destroy();
  }
}

function awsS3Options({
  url,
  method = "PUT",
  headers = {},
  multipart,
  file,
}: DirectUploadRequest): AwsS3Options<UploadMeta, UploadBody> {
  const getUploadParameters = (): AwsS3UploadParameters => ({
    method: uploadMethod(method),
    url,
    headers,
  });

  if (!multipart) {
    return {
      allowedMetaFields: false,
      shouldUseMultipart: false,
      getUploadParameters,
    };
  }

  return {
    allowedMetaFields: false,
    getUploadParameters,
    createMultipartUpload: () => multipart.createMultipartUpload(file),
    listParts: (_uppyFile, upload) => multipart.listParts?.(file, upload) ?? [],
    signPart: (_uppyFile, part) => multipart.signPart(file, part),
    completeMultipartUpload: async (_uppyFile, upload) =>
      (await multipart.completeMultipartUpload(file, upload)) ?? {},
    abortMultipartUpload: (_uppyFile, upload) => multipart.abortMultipartUpload(file, upload),
  };
}

function uploadProgress(file: File, loaded: number, total: number | null): UploadProgress {
  const safeTotal = total ?? file.size;
  return {
    loaded,
    total: safeTotal,
    percent: safeTotal > 0 ? Math.round((loaded / safeTotal) * 100) : 0,
  };
}

function uploadMethod(method: string): UploadMethod {
  const normalized = method.toUpperCase();
  if (normalized === "PUT") {
    return normalized;
  }
  throw new Error(`Upload method ${method} is not supported.`);
}

function cancelledError() {
  return new Error("Upload cancelled.");
}

function isAbortError(error: unknown) {
  return error instanceof Error && error.name === "AbortError";
}
