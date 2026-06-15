import { type MutationKey, useMutation } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { toast } from "sonner";

import {
  type DirectMultipartUploadRequest,
  type UploadProgress,
  type UploadTransport,
  uploadWithProgress,
} from "@/lib/direct-upload";

type UploadText = string | ((file: File) => string);
type UploadErrorSurface = "toast" | "inline";

interface DirectUploadIntentRequest {
  url: string;
  transport: UploadTransport;
  method?: string;
  headers?: Record<string, string>;
  multipart?: DirectMultipartUploadRequest;
}

interface DirectUploadOptions<TIntent, TResult, TVars extends { file: File }> {
  mutationKey: MutationKey;
  createIntent: (vars: TVars) => Promise<TIntent>;
  uploadRequest: (intent: TIntent, vars: TVars) => DirectUploadIntentRequest;
  completeUpload: (intent: TIntent, vars: TVars) => Promise<TResult>;
  loadingText?: UploadText;
  successText?: UploadText;
  errorText?: UploadText;
  errorSurface?: UploadErrorSurface;
}

export function useDirectUpload<TIntent, TResult, TVars extends { file: File } = { file: File }>({
  mutationKey,
  createIntent,
  uploadRequest,
  completeUpload,
  loadingText,
  successText,
  errorText,
  errorSurface = "toast",
}: DirectUploadOptions<TIntent, TResult, TVars>) {
  const [progress, setProgress] = useState<UploadProgress | null>(null);
  const lastToastPercent = useRef<number | null>(null);

  const mutation = useMutation<TResult, Error, TVars>({
    mutationKey,
    onError: () => undefined,
    mutationFn: async (vars) => {
      const { file } = vars;
      lastToastPercent.current = null;
      setProgress({ loaded: 0, total: file.size, percent: 0 });

      const loadingTitle = uploadText(loadingText, file, "Uploading");
      const toastID = toast.loading(loadingTitle, { description: "Preparing upload" });

      try {
        const intent = await createIntent(vars);
        toast.loading(loadingTitle, { id: toastID, description: "0%" });
        await uploadWithProgress({
          ...uploadRequest(intent, vars),
          file,
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
        const result = await completeUpload(intent, vars);
        toast.success(uploadText(successText, file, "Upload complete"), { id: toastID });
        return result;
      } catch (error) {
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
        lastToastPercent.current = null;
        setProgress(null);
      }
    },
  });

  return {
    progress,
    mutation,
    upload: mutation.mutateAsync,
    isUploading: mutation.isPending,
    error: mutation.error,
    reset: mutation.reset,
  };
}

function uploadText(text: UploadText | undefined, file: File, fallback: string) {
  return typeof text === "function" ? text(file) : (text ?? fallback);
}
