import { HostImage } from "@/components/hosts/host-image";
import { HostStatus } from "@/components/hosts/host-status";
import type { Host } from "@/hooks/use-hosts";
import { formatRelative } from "@/lib/utils";

export function HostHeader({ host }: { host: Host }) {
  return (
    <div className="flex items-center gap-4">
      <div className="shrink-0">
        <HostImage hardwareModel={host.hardware.model_identifier} />
      </div>
      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <h1 className="text-foreground min-w-0 truncate text-xl font-semibold" title={host.display_name}>
            {host.display_name}
          </h1>
          <HostStatus status={host.status} />
        </div>
        <p className="text-muted-foreground text-xs">
          {host.timestamps.inventory_updated_at ? (
            <span title={new Date(host.timestamps.inventory_updated_at).toLocaleString()}>
              Last fetched {formatRelative(host.timestamps.inventory_updated_at)}
            </span>
          ) : (
            "Never fetched"
          )}
        </p>
      </div>
    </div>
  );
}
