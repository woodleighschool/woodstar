import type { HostStatus } from "@/components/hosts/host-status";
import { cn } from "@/lib/utils";

export function HostStatusPill({ status, className }: { status: HostStatus; className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium",
        status === "online"
          ? "border-status-online/30 bg-status-online/10 text-status-online"
          : "border-border bg-muted text-muted-foreground",
        className,
      )}
    >
      <span className={cn("size-1.5 rounded-full", status === "online" ? "bg-status-online" : "bg-status-offline")} />
      {status === "online" ? "Online" : "Offline"}
    </span>
  );
}
