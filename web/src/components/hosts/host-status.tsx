import { Status, StatusIndicator, StatusLabel } from "@/components/ui/status";
import type { Host } from "@/hooks/use-hosts";

type HostStatusValue = Host["status"];

export function HostStatus({ status }: { status: HostStatusValue }) {
  const online = status === "online";

  return (
    <Status variant={online ? "success" : "default"}>
      <StatusIndicator />
      <StatusLabel>{online ? "Online" : "Offline"}</StatusLabel>
    </Status>
  );
}
