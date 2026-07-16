import { useEffect } from "react";

export const clientResourceImageAccept = "image/jpeg,image/png";
export const clientResourceImageMaxSize = 5 * 1024 * 1024;

export interface ClientResourceAsset {
  name: string;
  url: string;
  objectID: number | null;
  file: File | null;
}

export function validateClientResourceImage(file: File) {
  if (file.size <= 0 || file.size > clientResourceImageMaxSize) {
    return "Image must be 5 MB or smaller.";
  }
  return null;
}

export function useClientResourceAsset(asset: ClientResourceAsset | null) {
  useEffect(
    () => () => {
      if (asset?.file) URL.revokeObjectURL(asset.url);
    },
    [asset],
  );

  return (file: File): ClientResourceAsset => ({
    name: file.name,
    url: URL.createObjectURL(file),
    objectID: null,
    file,
  });
}
