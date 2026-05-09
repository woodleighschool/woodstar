import { useNavigate, useSearch } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";

/**
 * Two-way binding between a free-text search-param and a debounced URL write.
 *
 * - `draft` updates immediately for input value (no jank).
 * - URL write happens after `debounceMs`.
 * - URL > draft sync happens automatically when the search param changes from elsewhere
 *   (e.g. router navigation, browser back).
 * - Empty/whitespace values clear the param (omitted from URL).
 *
 * Resetting `page` to undefined alongside the search write is the common case.
 */
export function useDebouncedSearchParam<TKey extends string>(
  key: TKey,
  options: { debounceMs?: number; resetKeys?: readonly string[] } = {},
): [string, (next: string) => void] {
  const debounceMs = options.debounceMs ?? 200;
  const resetKeys = options.resetKeys ?? ["page"];

  const search: Record<string, unknown> = useSearch({ strict: false });
  const navigate = useNavigate();
  const raw = search[key];
  const urlValue = typeof raw === "string" ? raw : "";

  const [draft, setDraft] = useState(urlValue);
  const [prevUrlValue, setPrevUrlValue] = useState(urlValue);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Sync URL > draft when the URL value changes externally (browser back, navigation).
  if (prevUrlValue !== urlValue) {
    setPrevUrlValue(urlValue);
    setDraft(urlValue);
  }

  useEffect(
    () => () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    },
    [],
  );

  const writeToUrl = (next: string) => {
    const trimmed = next.trim();
    void navigate({
      search: ((prev: Record<string, unknown>) => {
        const out: Record<string, unknown> = { ...prev };
        if (trimmed === "") delete out[key];
        else out[key] = trimmed;
        for (const r of resetKeys) delete out[r];
        return out;
      }) as never,
      replace: true,
    });
  };

  const setValue = (next: string) => {
    setDraft(next);
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => writeToUrl(next), debounceMs);
  };

  return [draft, setValue];
}
