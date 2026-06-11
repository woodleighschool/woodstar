import type { SortingState } from "@tanstack/react-table";
import { createParser } from "nuqs/server";

interface SortingItem {
  id: string;
  desc: boolean;
}

export const getSortingStateParser = (columnIds?: string[] | Set<string>) => {
  const validKeys = columnIds ? (columnIds instanceof Set ? columnIds : new Set(columnIds)) : null;

  return createParser({
    parse: (value) => {
      const parsed = parseSortingState(value);
      if (parsed === null) return null;

      if (validKeys && parsed.some((item) => !validKeys.has(item.id))) {
        return null;
      }

      return parsed;
    },
    serialize: serializeSortingState,
    eq: (a, b) =>
      a.length === b.length && a.every((item, index) => item.id === b[index]?.id && item.desc === b[index]?.desc),
  });
};

export function serializeSortingState(value: readonly SortingItem[]): string {
  const sort = value[0];
  return sort ? `${sort.id}.${sort.desc ? "desc" : "asc"}` : "";
}

function parseSortingState(value: string): SortingState | null {
  const trimmed = value.trim();
  if (trimmed === "") return [];
  if (trimmed.includes(",")) return null;

  const item = parseSortToken(trimmed);
  return item === null ? null : [item];
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
