import { HostImage } from "@/components/hosts/host-image";
import { statusFromHost } from "@/components/hosts/host-status";
import { HostStatusPill } from "@/components/hosts/host-status-pill";
import type { Host } from "@/hooks/use-hosts";
import { formatRelative } from "@/lib/utils";

export function HostHeader({ host }: { host: Host }) {
  const displayName = host.display_name || host.hardware_uuid;

  return (
    <div className="flex items-center gap-4">
      <HostImage hardwareModel={host.hardware_model} />
      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <h1 className="text-foreground truncate text-xl font-semibold" title={displayName}>
            {displayName}
          </h1>
          <HostStatusPill status={statusFromHost(host)} />
        </div>
        <p className="text-muted-foreground text-xs">
          {host.detail_updated_at ? (
            <span title={new Date(host.detail_updated_at).toLocaleString()}>
              Last fetched {formatRelative(host.detail_updated_at)}
            </span>
          ) : (
            "Never fetched"
          )}
        </p>
      </div>
    </div>
  );
}
