export interface UploadProgress {
  loaded: number;
  total: number;
  percent: number;
}

interface UploadTarget {
  url: string;
  method: "PUT";
  headers: Record<string, string>;
}

export interface MultipartUploadRequest {
  signPart: (partNumber: number, signal?: AbortSignal) => Promise<UploadTarget>;
}

export type UploadRequest =
  | ({ strategy: "direct-put" } & UploadTarget)
  | {
      strategy: "multipart";
      multipart: MultipartUploadRequest;
    };

export interface CompletedUploadPart {
  partNumber: number;
  etag: string;
}

export type UploadResult =
  | { strategy: "direct-put" }
  | { strategy: "multipart"; parts: CompletedUploadPart[] };

type UploadContext = {
  file: File;
  signal?: AbortSignal;
  onProgress?: (progress: UploadProgress) => void;
};

export type UploadExecution = UploadRequest & UploadContext;

const multipartPartSize = 64 * 1024 * 1024;
const maximumMultipartParts = 10_000;

export async function uploadWithProgress(request: UploadExecution): Promise<UploadResult> {
  if (request.strategy === "direct-put") {
    await uploadTarget(
      request,
      request.file,
      request.file.size,
      0,
      request.onProgress,
      request.signal,
    );
    return { strategy: "direct-put" };
  }
  return uploadWithMultipartProgress(request);
}

async function uploadWithMultipartProgress(
  request: Extract<UploadExecution, { strategy: "multipart" }>,
): Promise<UploadResult> {
  const { file, signal, onProgress } = request;
  throwIfCancelled(signal);

  const partSize = Math.max(multipartPartSize, Math.ceil(file.size / maximumMultipartParts));
  const parts: CompletedUploadPart[] = [];
  let completedBytes = 0;

  for (let offset = 0, partNumber = 1; offset < file.size; offset += partSize, partNumber++) {
    const chunk = file.slice(offset, Math.min(offset + partSize, file.size));
    const etag = await uploadPart(
      request.multipart,
      partNumber,
      chunk,
      file.size,
      completedBytes,
      onProgress,
      signal,
    );
    parts.push({ partNumber, etag });
    completedBytes += chunk.size;
    onProgress?.(uploadProgress(completedBytes, file.size));
  }

  return { strategy: "multipart", parts };
}

async function uploadPart(
  multipart: MultipartUploadRequest,
  partNumber: number,
  chunk: Blob,
  totalBytes: number,
  completedBytes: number,
  onProgress?: (progress: UploadProgress) => void,
  signal?: AbortSignal,
) {
  let lastError: unknown;
  for (let attempt = 0; attempt < 2; attempt++) {
    throwIfCancelled(signal);
    const target = await multipart.signPart(partNumber, signal);
    try {
      const etag = await uploadTarget(
        target,
        chunk,
        totalBytes,
        completedBytes,
        onProgress,
        signal,
      );
      if (!etag) {
        throw new Error(`Multipart part ${partNumber} did not return an ETag.`);
      }
      return etag;
    } catch (error) {
      if (signal?.aborted) throw cancelledError();
      lastError = error;
      onProgress?.(uploadProgress(completedBytes, totalBytes));
    }
  }
  throw lastError;
}

function uploadTarget(
  target: UploadTarget,
  body: Blob,
  totalBytes: number,
  completedBytes: number,
  onProgress?: (progress: UploadProgress) => void,
  signal?: AbortSignal,
) {
  return new Promise<string | null>((resolve, reject) => {
    if (signal?.aborted) {
      reject(cancelledError());
      return;
    }

    const xhr = new XMLHttpRequest();
    const finish = () => signal?.removeEventListener("abort", abort);
    const abort = () => xhr.abort();

    xhr.upload.addEventListener("progress", (event) => {
      onProgress?.(uploadProgress(completedBytes + event.loaded, totalBytes));
    });
    xhr.addEventListener("load", () => {
      finish();
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(xhr.getResponseHeader("ETag"));
        return;
      }
      reject(new Error(`Upload failed with HTTP ${xhr.status}.`));
    });
    xhr.addEventListener("error", () => {
      finish();
      reject(new Error("Upload failed before the storage service accepted the request."));
    });
    xhr.addEventListener("abort", () => {
      finish();
      reject(cancelledError());
    });

    signal?.addEventListener("abort", abort, { once: true });
    xhr.open(target.method, target.url);
    for (const [key, value] of Object.entries(target.headers)) {
      xhr.setRequestHeader(key, value);
    }
    xhr.send(body);
  });
}

function uploadProgress(loaded: number, total: number): UploadProgress {
  return {
    loaded,
    total,
    percent: total > 0 ? Math.round((loaded / total) * 100) : 0,
  };
}

function throwIfCancelled(signal?: AbortSignal) {
  if (signal?.aborted) throw cancelledError();
}

function cancelledError() {
  return new Error("Upload cancelled.");
}
