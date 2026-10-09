import { type MutationKey, useMutation } from "@tanstack/react-query";
import {
  type Content,
  type UploadProgress,
  type UploadRequest,
  digest,
  upload,
} from "@woodleighschool/bloby-client";
import { useRef, useState } from "react";

import { toast } from "@components/ui/toast";

type UploadText = string | ((file: File) => string);
type UploadErrorSurface = "toast" | "inline";

interface UploadOptions<TIntent, TResult, TVars extends { file: File }> {
  mutationKey: MutationKey;
  createIntent: (
    declaration: Content & { filename: string },
    signal: AbortSignal,
  ) => Promise<TIntent>;
  uploadRequest: (intent: TIntent, vars: TVars) => UploadRequest;
  completeUpload: (intent: TIntent, vars: TVars, signal: AbortSignal) => Promise<TResult>;
  onSuccess?: (result: TResult, vars: TVars) => void | Promise<void>;
  cleanupIntent?: (intent: TIntent, vars: TVars) => Promise<void>;
  loadingText?: UploadText;
  successText?: UploadText;
  errorText?: UploadText;
  errorSurface?: UploadErrorSurface;
}

export function useUpload<TIntent, TResult, TVars extends { file: File } = { file: File }>({
  mutationKey,
  createIntent,
  uploadRequest,
  completeUpload,
  onSuccess,
  cleanupIntent,
  loadingText,
  successText,
  errorText,
  errorSurface = "toast",
}: UploadOptions<TIntent, TResult, TVars>) {
  const [statusText, setStatusText] = useState<string | null>(null);
  const uploadAbort = useRef<AbortController | null>(null);

  const mutation = useMutation<TResult, Error, TVars>({
    mutationKey,
    onError: () => undefined,
    onSuccess,
    mutationFn: async (vars) => {
      const { file } = vars;
      const abortController = new AbortController();
      uploadAbort.current = abortController;
      const { signal } = abortController;

      const toastID = toast.add({
        title: uploadText(loadingText, file, "Uploading"),
        type: "loading",
        timeout: 0,
      });
      let announced: string | undefined;
      const announce = (description: string) => {
        if (description === announced) return;
        announced = description;
        setStatusText(description);
        toast.update(toastID, { description });
      };
      const progress =
        (phase: string) =>
        ({ percent }: UploadProgress) =>
          announce(`${phase} ${percent}%`);

      let intent: TIntent | undefined;
      try {
        const content = await digest(file, {
          signal,
          onProgress: progress("Computing checksums"),
        });
        signal.throwIfAborted();
        intent = await createIntent({ filename: file.name, ...content }, signal);
        signal.throwIfAborted();
        await upload({
          ...uploadRequest(intent, vars),
          blob: file,
          signal,
          onProgress: progress("Uploading"),
        });
        signal.throwIfAborted();
        announce("Finalizing");
        const result = await completeUpload(intent, vars, signal);
        toast.update(toastID, {
          title: uploadText(successText, file, "Upload Complete"),
          description: undefined,
          type: "success",
          timeout: 5000,
        });
        return result;
      } catch (error) {
        // The server releases an upload it refuses to finalize, so the release
        // may find nothing left.
        if (intent !== undefined) await cleanupIntent?.(intent, vars).catch(() => undefined);
        if (signal.aborted || errorSurface === "inline") {
          toast.close(toastID);
        } else {
          toast.update(toastID, {
            title: uploadText(errorText, file, "Upload failed"),
            description: error instanceof Error ? error.message : "Unknown upload error.",
            type: "error",
            timeout: 5000,
          });
        }
        throw error;
      } finally {
        if (uploadAbort.current === abortController) {
          uploadAbort.current = null;
        }
        setStatusText(null);
      }
    },
  });

  return {
    statusText,
    mutation,
    upload: mutation.mutateAsync,
    cancel: () => uploadAbort.current?.abort(),
    isUploading: mutation.isPending,
    error: mutation.error,
    reset: mutation.reset,
  };
}

function uploadText(text: UploadText | undefined, file: File, fallback: string) {
  return typeof text === "function" ? text(file) : (text ?? fallback);
}
