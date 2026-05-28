import type { HostStatus } from "@/components/hosts/host-status";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export function HostStatusPill({ status, className }: { status: HostStatus; className?: string }) {
  return (
    <Badge variant={status === "online" ? "success" : "secondary"} className={cn("gap-1.5", className)}>
      <span className={cn("size-1.5 rounded-full", status === "online" ? "bg-status-online" : "bg-status-offline")} />
      {status === "online" ? "Online" : "Offline"}
    </Badge>
  );
}
