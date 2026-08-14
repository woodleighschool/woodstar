import { Laptop } from "lucide-react";
import type { ReactNode } from "react";

import { PageHeader } from "@components/layout/page-layout";
import { HostOnlineDot } from "@features/hosts/components/host-online-dot";
import type { Host } from "@lib/api";
import { formatRelative } from "@lib/utils";

export function HostHeader({ host, actions }: { host: Host; actions?: ReactNode }) {
  return (
    <PageHeader
      title={host.display_name}
      icon={<Laptop />}
      context={<HostOnlineDot status={host.status} animated />}
      actions={actions}
      meta={
        host.inventory_updated_at ? (
          <span title={new Date(host.inventory_updated_at).toLocaleString()}>
            Last fetched {formatRelative(host.inventory_updated_at)}
          </span>
        ) : (
          "Never fetched"
        )
      }
    />
  );
}
