import { HostImage } from "@/components/hosts/host-image";
import { statusFromHost } from "@/components/hosts/host-status";
import type { Host } from "@/hooks/use-hosts";
import { formatRelative } from "@/lib/utils";

export function HostHeader({ host }: { host: Host }) {
  const displayName = host.display_name || host.hardware_uuid;
  const isOnline = statusFromHost(host) === "online";

  return (
    <div className="flex items-center gap-4">
      <div className="relative shrink-0">
        <HostImage hardwareModel={host.hardware_model} />
        {isOnline ? (
          <span
            aria-label="Online"
            title="Online"
            className="bg-status-online ring-background absolute top-2 right-2 size-2.5 rounded-full ring-2"
          />
        ) : null}
      </div>
      <div className="flex min-w-0 flex-col gap-1">
        <h1 className="text-foreground truncate text-xl font-semibold" title={displayName}>
          {displayName}
        </h1>
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
