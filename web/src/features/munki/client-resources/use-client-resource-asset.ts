import { useEffect } from "react";

import type { ClientResourceAsset } from "./form-schema";

export const clientResourceImageAccept = "image/jpeg,image/png";
export const clientResourceImageMaxSize = 5 * 1024 * 1024;

export function validateClientResourceImage(file: File) {
  if (file.size <= 0 || file.size > clientResourceImageMaxSize) {
    return "Image must be 5 MB or smaller.";
  }
  return null;
}

export function useClientResourceAssetLifecycle(asset: ClientResourceAsset | null) {
  useEffect(
    () => () => {
      if (asset?.file) URL.revokeObjectURL(asset.url);
    },
    [asset],
  );
}

export function createClientResourceAsset(file: File): ClientResourceAsset {
  return {
    name: file.name,
    url: URL.createObjectURL(file),
    objectID: null,
    file,
  };
}
