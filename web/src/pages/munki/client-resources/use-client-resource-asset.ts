import { useEffect, useRef, useState } from "react";

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

export function useClientResourceAsset(initial: ClientResourceAsset | null) {
  const [replacement, setReplacement] = useState<ClientResourceAsset | null>(null);
  const [error, setError] = useState<string | null>(null);
  const ownedURLs = useRef(new Set<string>());
  const asset = replacement ?? initial;

  useEffect(
    () => () => {
      for (const url of ownedURLs.current) URL.revokeObjectURL(url);
    },
    [],
  );

  function replace(file: File) {
    const validationError = validateClientResourceImage(file);
    if (validationError) {
      setError(validationError);
      return false;
    }

    const url = URL.createObjectURL(file);
    if (replacement?.file) {
      URL.revokeObjectURL(replacement.url);
      ownedURLs.current.delete(replacement.url);
    }
    ownedURLs.current.add(url);
    setError(null);
    setReplacement({ name: file.name, url, objectID: null, file });
    return true;
  }

  function reset() {
    if (replacement?.file) {
      URL.revokeObjectURL(replacement.url);
      ownedURLs.current.delete(replacement.url);
    }
    setError(null);
    setReplacement(null);
  }

  return { asset, error, replace, reset };
}
