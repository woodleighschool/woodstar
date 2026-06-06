import { useParams } from "@tanstack/react-router";

export function useSoftwareIDParam() {
  const params = useParams({ strict: false });
  const id = Number(params.softwareId);
  return Number.isFinite(id) && id > 0 ? id : null;
}

export function usePackageIDParam() {
  const params = useParams({ strict: false });
  const id = Number(params.packageId);
  return Number.isFinite(id) && id > 0 ? id : null;
}

export function optionalText(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

export function uniqueOptions(values: string[]) {
  return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean))).sort((a, b) => a.localeCompare(b));
}
