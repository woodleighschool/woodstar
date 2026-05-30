export type HostStatus = "online" | "offline";

export function statusFromHost(args: { status?: string }): HostStatus {
  return args.status === "online" ? "online" : "offline";
}
