import type { Host } from "@lib/api";

export function HostOnlineDot({ status }: { status: Host["status"] }) {
  if (status !== "online") return null;

  return (
    <span className="size-2 shrink-0 rounded-full bg-success">
      <span className="sr-only">Osquery online</span>
    </span>
  );
}
