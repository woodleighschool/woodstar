import { Laptop } from "lucide-react";

import { PageHeader } from "@components/layout/page-layout";
import { HostStatus } from "@features/hosts/components/host-status";
import type { Host } from "@lib/api";
import { formatRelative } from "@lib/utils";

export function HostHeader({ host }: { host: Host }) {
  return (
    <PageHeader
      title={host.display_name}
      icon={<Laptop />}
      context={<HostStatus status={host.status} />}
      meta={
        host.timestamps.inventory_updated_at ? (
          <span title={new Date(host.timestamps.inventory_updated_at).toLocaleString()}>
            Last fetched {formatRelative(host.timestamps.inventory_updated_at)}
          </span>
        ) : (
          "Never fetched"
        )
      }
    />
  );
}
