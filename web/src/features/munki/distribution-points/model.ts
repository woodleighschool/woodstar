import type { MunkiDistributionPointWorker, MunkiPackageState } from "@lib/api";
import { enumLabel, type EnumMetadataMap, type StatusMetadataMap } from "@lib/enum-metadata";

export type DistributionPointWorkerStatus = "offline" | "online" | "incompatible";

export const DISTRIBUTION_POINT_WORKER_STATUSES = {
  offline: { name: "Offline" },
  online: { name: "Online", variant: "success" },
  incompatible: {
    name: "Incompatible",
    description: "This worker does not support the server's protocol version.",
    variant: "warning",
  },
} satisfies StatusMetadataMap<DistributionPointWorkerStatus>;

const PACKAGE_STATES = {
  pending: { name: "Pending" },
  syncing: { name: "Syncing" },
  current: { name: "Current" },
  error: { name: "Error" },
} satisfies EnumMetadataMap<MunkiPackageState["status"]>;

export function distributionPointWorkerStatus(
  worker?: MunkiDistributionPointWorker,
): DistributionPointWorkerStatus {
  if (!worker) return "offline";
  return worker.compatible ? "online" : "incompatible";
}

export function packageStateLabel(status: MunkiPackageState["status"]) {
  return enumLabel(PACKAGE_STATES, status);
}
