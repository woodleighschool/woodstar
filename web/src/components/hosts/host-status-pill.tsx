import { cn } from "@/lib/utils";

type Status = "online" | "offline";

export function statusFromHost(args: { last_seen_at?: string | null }): Status {
  if (!args.last_seen_at) return "offline";
  const lastSeen = new Date(args.last_seen_at).getTime();
  return Date.now() - lastSeen <= 5 * 60 * 1000 ? "online" : "offline";
}

export function HostStatusPill({ status, className }: { status: Status; className?: string }) {
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
