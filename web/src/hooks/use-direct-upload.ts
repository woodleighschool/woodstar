import { type MutationKey, useMutation } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { toast } from "sonner";

import { type UploadProgress, uploadWithProgress } from "@/lib/direct-upload";

type UploadText = string | ((file: File) => string);
type UploadErrorSurface = "toast" | "inline";

interface DirectUploadIntentRequest {
  url: string;
  method?: "PUT";
  headers?: Record<string, string>;
}

interface DirectUploadOptions<TIntent, TResult> {
  mutationKey: MutationKey;
  createIntent: (file: File) => Promise<TIntent>;
  uploadRequest: (intent: TIntent, file: File) => DirectUploadIntentRequest;
  completeUpload: (intent: TIntent, file: File) => Promise<TResult>;
  loadingText?: UploadText;
  successText?: UploadText;
  errorText?: UploadText;
  errorSurface?: UploadErrorSurface;
}

export function useDirectUpload<TIntent, TResult>({
  mutationKey,
  createIntent,
  uploadRequest,
  completeUpload,
  loadingText,
  successText,
  errorText,
  errorSurface = "toast",
}: DirectUploadOptions<TIntent, TResult>) {
  const [progress, setProgress] = useState<UploadProgress | null>(null);
  const lastToastPercent = useRef<number | null>(null);

  const mutation = useMutation<TResult, Error, File>({
    mutationKey,
    onError: () => undefined,
    mutationFn: async (file) => {
      lastToastPercent.current = null;
      setProgress({ loaded: 0, total: file.size, percent: 0 });

      const loadingTitle = uploadText(loadingText, file, "Uploading");
      const toastID = toast.loading(loadingTitle, { description: "Preparing upload" });

      try {
        const intent = await createIntent(file);
        toast.loading(loadingTitle, { id: toastID, description: "0%" });
        await uploadWithProgress({
          ...uploadRequest(intent, file),
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
        const result = await completeUpload(intent, file);
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
