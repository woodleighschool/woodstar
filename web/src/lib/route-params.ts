export function parseRouteID(value: string | null | undefined): number | null {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

export function detailPath(id: number | null): { id: number } {
  if (id === null) {
    throw new Error("detail query ran without an id");
  }
  return { id };
}
