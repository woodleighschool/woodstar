import { cn } from "@/lib/utils";

export type HostStatus = "online" | "offline" | "error";

const colors: Record<HostStatus, string> = {
  online: "bg-status-online",
  offline: "bg-status-offline",
  error: "bg-status-error",
};

export function StatusDot({
  status,
  className,
}: {
  status: HostStatus;
  className?: string;
}) {
  return (
    <span
      className={cn("inline-block size-2 rounded-full", colors[status], className)}
      aria-label={status}
    />
  );
}
