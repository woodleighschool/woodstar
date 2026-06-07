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
