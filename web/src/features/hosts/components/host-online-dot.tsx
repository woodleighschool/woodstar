import { StatusIndicator } from "@components/ui/status";
import type { Host } from "@lib/api";
import { cn } from "@lib/utils";

export function HostOnlineDot({
  status,
  animated = false,
}: {
  status: Host["status"];
  animated?: boolean;
}) {
  if (status !== "online") return null;

  return (
    <StatusIndicator aria-hidden className={cn("bg-success", !animated && "before:animate-none")} />
  );
}
