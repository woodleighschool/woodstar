export type HostStatus = "online" | "offline";

export function statusFromHost(args: { last_seen_at?: string | null }): HostStatus {
  if (!args.last_seen_at) return "offline";
  const lastSeen = new Date(args.last_seen_at).getTime();
  return Date.now() - lastSeen <= 5 * 60 * 1000 ? "online" : "offline";
}
