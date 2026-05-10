import { useEffect, useState } from "react";

function readInitial(key: string, fallback: string[]): string[] {
  if (typeof window === "undefined") return fallback;
  const raw = window.localStorage.getItem(key);
  if (raw === null) return fallback;
  try {
    const parsed: unknown = JSON.parse(raw);
    if (Array.isArray(parsed)) return parsed.filter((s): s is string => typeof s === "string");
  } catch {
    // fall through
  }
  return fallback;
}

export function useHiddenColumns(storageKey: string, defaultHidden: string[] = []) {
  const [hidden, setHidden] = useState<string[]>(() => readInitial(storageKey, defaultHidden));

  useEffect(() => {
    window.localStorage.setItem(storageKey, JSON.stringify(hidden));
  }, [storageKey, hidden]);

  function toggle(id: string) {
    setHidden((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  }

  return { hidden, toggle, setHidden } as const;
}
