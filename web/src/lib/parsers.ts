import type { SortingState } from "@tanstack/react-table";

interface SortingItem {
  id: string;
  desc: boolean;
}

export function serializeSortingState(value: readonly SortingItem[]): string {
  const sort = value[0];
  return sort ? `${sort.id}.${sort.desc ? "desc" : "asc"}` : "";
}

export function parseSortingState(value: string | undefined): SortingState {
  if (value === undefined) return [];

  const trimmed = value.trim();
  if (trimmed === "") return [];
  if (trimmed.includes(",")) return [];

  const item = parseSortToken(trimmed);
  return item === null ? [] : [item];
}

function parseSortToken(token: string): SortingItem | null {
  if (token === "") return null;

  const dot = token.lastIndexOf(".");
  if (dot === -1) return { id: token, desc: false };

  const id = token.slice(0, dot);
  const direction = token.slice(dot + 1);
  if (direction === "asc") return id ? { id, desc: false } : null;
  if (direction === "desc") return id ? { id, desc: true } : null;

  return { id: token, desc: false };
}
