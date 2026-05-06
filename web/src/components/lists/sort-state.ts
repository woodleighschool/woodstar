export type SortDirection = "asc" | "desc";

export interface SortState {
  orderKey?: string;
  orderDirection?: SortDirection;
}

export function nextSortState(current: SortState, orderKey: string): SortState {
  if (current.orderKey !== orderKey) {
    return { orderKey, orderDirection: "asc" };
  }
  if (current.orderDirection === "asc") {
    return { orderKey, orderDirection: "desc" };
  }
  return {};
}
