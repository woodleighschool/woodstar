import { Badge } from "@/components/ui/badge";
import { Status, StatusIndicator, StatusLabel } from "@/components/ui/status";
import type { MunkiPackageState } from "@/lib/api";

export function ConnectionBadge({ online }: { online: boolean }) {
  return (
    <Status variant={online ? "success" : "default"}>
      <StatusIndicator className={online ? undefined : "before:hidden"} />
      <StatusLabel>{online ? "Online" : "Offline"}</StatusLabel>
    </Status>
  );
}

export function MirrorBadge({ packages }: { packages: MunkiPackageState[] }) {
  if (packages.length === 0) return <span className="text-muted-foreground">-</span>;

  const failed = packages.some((pkg) => pkg.status === "error");
  const pending = packages.some((pkg) => pkg.status === "pending");
  const syncing = packages.some((pkg) => pkg.status === "syncing");
  const current = !failed && !pending && !syncing;

  if (failed) {
    return (
      <Status variant="error">
        <StatusIndicator className="before:hidden" />
        <StatusLabel>Error</StatusLabel>
      </Status>
    );
  }

  return (
    <Status variant={current ? "success" : syncing ? "info" : "default"}>
      <StatusIndicator className="before:hidden" />
      <StatusLabel>{current ? "Current" : syncing ? "Syncing" : "Pending"}</StatusLabel>
    </Status>
  );
}

export function PackageStatusBadge({ status }: { status: MunkiPackageState["status"] }) {
  const current = status === "current";
  const syncing = status === "syncing";
  const failed = status === "error";
  return (
    <Status variant={current ? "success" : syncing ? "info" : failed ? "error" : "default"}>
      <StatusIndicator className="before:hidden" />
      <StatusLabel>
        {current ? "Current" : syncing ? "Syncing" : failed ? "Error" : "Pending"}
      </StatusLabel>
    </Status>
  );
}

export function BoolBadge({ value, label }: { value: boolean; label: string }) {
  return (
    <Badge variant={value ? "secondary" : "outline"} className="font-normal">
      {value ? label : "No"}
    </Badge>
  );
}
