export interface UploadProgress {
  loaded: number;
  total: number;
  percent: number;
}

export interface DirectUploadRequest {
  url: string;
  file: File;
  method?: "PUT";
  headers?: Record<string, string>;
  signal?: AbortSignal;
  onProgress?: (progress: UploadProgress) => void;
}

export function uploadWithProgress({
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
      const percent = event.lengthComputable && total > 0 ? Math.round((event.loaded / total) * 100) : 0;
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

export async function fileSHA256(file: File) {
  const digest = await crypto.subtle.digest("SHA-256", await file.arrayBuffer());
  return Array.from(new Uint8Array(digest))
    .map((byte) => byte.toString(16).padStart(2, "0"))
    .join("");
}
