import { type MutationKey, useMutation } from "@tanstack/react-query";
import { useRef, useState } from "react";

import { toast } from "@components/ui/toast";
import {
  type UploadProgress,
  type UploadRequest,
  type UploadResult,
  uploadWithProgress,
} from "@lib/upload";

type UploadText = string | ((file: File) => string);
type UploadErrorSurface = "toast" | "inline";

interface UploadOptions<TIntent, TResult, TVars extends { file: File }> {
  mutationKey: MutationKey;
  createIntent: (vars: TVars) => Promise<TIntent>;
  uploadRequest: (intent: TIntent, vars: TVars) => UploadRequest;
  completeUpload: (
    intent: TIntent,
    vars: TVars,
    transfer: UploadResult,
    signal: AbortSignal,
  ) => Promise<TResult>;
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
      const toastID = toast.add({
        title: loadingTitle,
        description: "Preparing upload",
        type: "loading",
        timeout: 0,
      });

      let intent: TIntent | undefined;
      let transfer: UploadResult | undefined;
      const finalize = async (uploadIntent: TIntent, uploadResult: UploadResult) => {
        setProgress({ loaded: file.size, total: file.size, percent: 100 });
        toast.update(toastID, {
          title: loadingTitle,
          description: "Finalizing",
          type: "loading",
          timeout: 0,
        });
        const result = await completeUpload(
          uploadIntent,
          vars,
          uploadResult,
          abortController.signal,
        );
        toast.update(toastID, {
          title: uploadText(successText, file, "Upload Complete"),
          description: undefined,
          type: "success",
          timeout: 5000,
        });
        return result;
      };
      try {
        intent = await createIntent(vars);
        toast.update(toastID, {
          title: loadingTitle,
          description: "0%",
          type: "loading",
          timeout: 0,
        });
        const request = uploadRequest(intent, vars);
        transfer = await uploadWithProgress({
          ...request,
          file,
          signal: abortController.signal,
          onProgress: (next) => {
            setProgress(next);
            if (lastToastPercent.current === next.percent) return;
            lastToastPercent.current = next.percent;
            toast.update(toastID, {
              title: loadingTitle,
              description: next.percent > 0 ? `${next.percent}%` : "Uploading",
              type: "loading",
              timeout: 0,
            });
          },
        });
        return await finalize(intent, transfer);
      } catch (error) {
        if (intent !== undefined && transfer === undefined) {
          await cleanupIntent?.(intent, vars).catch(() => undefined);
        }
        if (errorSurface === "toast") {
          toast.update(toastID, {
            title: uploadText(errorText, file, "Upload failed"),
            description: error instanceof Error ? error.message : "Unknown upload error.",
            type: "error",
            timeout: 5000,
          });
        } else {
          toast.close(toastID);
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
