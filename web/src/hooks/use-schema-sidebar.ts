import { useEffect, useState } from "react";

const STORAGE_KEY = "woodstar:schema-sidebar-open";

function readInitial(): boolean {
  if (typeof window === "undefined") return true;
  const raw = window.localStorage.getItem(STORAGE_KEY);
  return raw === null ? true : raw === "1";
}

export function useSchemaSidebar() {
  const [open, setOpen] = useState<boolean>(readInitial);

  useEffect(() => {
    window.localStorage.setItem(STORAGE_KEY, open ? "1" : "0");
  }, [open]);

  return [open, setOpen] as const;
}
