import { type MutationKey, useMutation } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { toast } from "sonner";

import { type UploadProgress, type UploadRequest, uploadWithProgress } from "@/lib/upload";

type UploadText = string | ((file: File) => string);
type UploadErrorSurface = "toast" | "inline";

interface UploadOptions<TIntent, TResult, TVars extends { file: File }> {
  mutationKey: MutationKey;
  createIntent: (vars: TVars) => Promise<TIntent>;
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
  const [progress, setProgress] = useState<UploadProgress | null>(null);
  const lastToastPercent = useRef<number | null>(null);
  const uploadAbort = useRef<AbortController | null>(null);

  const mutation = useMutation<TResult, Error, TVars>({
    mutationKey,
    onError: () => undefined,
    onSuccess,
    mutationFn: async (vars) => {
      const { file } = vars;
      const abortController = new AbortController();
      uploadAbort.current = abortController;
      lastToastPercent.current = null;
      setProgress({ loaded: 0, total: file.size, percent: 0 });

      const loadingTitle = uploadText(loadingText, file, "Uploading");
      const toastID = toast.loading(loadingTitle, { description: "Preparing upload" });

      let intent: TIntent | undefined;
      try {
        intent = await createIntent(vars);
        toast.loading(loadingTitle, { id: toastID, description: "0%" });
        await uploadWithProgress({
          ...uploadRequest(intent, vars),
          file,
          signal: abortController.signal,
          onProgress: (next) => {
            setProgress(next);
            if (lastToastPercent.current === next.percent) return;
            lastToastPercent.current = next.percent;
            toast.loading(loadingTitle, {
              id: toastID,
              description: next.percent > 0 ? `${next.percent}%` : "Uploading",
            });
          },
        });
        setProgress({ loaded: file.size, total: file.size, percent: 100 });
        toast.loading(loadingTitle, { id: toastID, description: "Finalizing" });
        const result = await completeUpload(intent, vars, abortController.signal);
        toast.success(uploadText(successText, file, "Upload complete"), { id: toastID });
        return result;
      } catch (error) {
        if (intent !== undefined) {
          await cleanupIntent?.(intent, vars).catch(() => undefined);
        }
        if (errorSurface === "toast") {
          toast.error(uploadText(errorText, file, "Upload failed"), {
            id: toastID,
            description: error instanceof Error ? error.message : "Unknown upload error.",
          });
        } else {
          toast.dismiss(toastID);
        }
        throw error;
      } finally {
        if (uploadAbort.current === abortController) {
          uploadAbort.current = null;
        }
        lastToastPercent.current = null;
        setProgress(null);
      }
    },
  });

  return {
    progress,
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
